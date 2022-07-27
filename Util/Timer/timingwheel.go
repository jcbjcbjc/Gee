package Timer

import (
	"errors"
	"sync/atomic"
	"time"
	"unsafe"

	"Gee/Util/Timer/delayqueue"
)

// TimingWheel is an implementation of Hierarchical Timing Wheels.
type TimingWheel struct {
	// 时间跨度,单位是毫秒
	tick int64 // in milliseconds
	// 时间轮个数
	wheelSize int64
	// 总跨度
	interval int64 // in milliseconds
	// 当前指针指向时间
	currentTime int64 // in milliseconds
	// 时间格列表
	buckets []*bucket
	// 延迟队列
	queue *delayqueue.DelayQueue

	// The higher-level overflow wheel.
	//
	// NOTE: This field may be updated and read concurrently, through Add().
	// 上级的时间轮饮用
	overflowWheel unsafe.Pointer // type: *TimingWheel

	exitC     chan struct{}
	waitGroup waitGroupWrapper
}

// NewTimingWheel creates an instance of TimingWheel with the given tick and wheelSize.
func NewTimingWheel(tick time.Duration, wheelSize int64) *TimingWheel {
	tickMs := int64(tick / time.Millisecond)
	if tickMs <= 0 {
		panic(errors.New("tick must be greater than or equal to 1ms"))
	}

	startMs := timeToMs(time.Now().UTC())

	return newTimingWheel(
		tickMs,
		wheelSize,
		startMs,
		delayqueue.New(int(wheelSize)),
	)
}

// newTimingWheel is an internal helper function that really creates an instance of TimingWheel.
func newTimingWheel(tickMs int64, wheelSize int64, startMs int64, queue *delayqueue.DelayQueue) *TimingWheel {
	buckets := make([]*bucket, wheelSize)
	for i := range buckets {
		buckets[i] = newBucket()
	}
	return &TimingWheel{
		tick:        tickMs,
		wheelSize:   wheelSize,
		currentTime: truncate(startMs, tickMs),
		interval:    tickMs * wheelSize,
		buckets:     buckets,
		queue:       queue,
		exitC:       make(chan struct{}),
	}
}

// add inserts the timer t into the current timing wheel.
func (tw *TimingWheel) add(t *Timer) bool {
	currentTime := atomic.LoadInt64(&tw.currentTime)
	// 已经过期
	if t.expiration < currentTime+tw.tick {
		// Already expired
		return false
		// 	到期时间在第一层环内
	} else if t.expiration < currentTime+tw.interval {
		// Put it into its own bucket
		// 获取时间轮的位置
		virtualID := t.expiration / tw.tick
		b := tw.buckets[virtualID%tw.wheelSize]
		// 将任务放入到bucket队列中
		b.Add(t)

		// Set the bucket expiration time
		// 如果是相同的时间，那么返回false，防止被多次插入到队列中
		if b.SetExpiration(virtualID * tw.tick) {
			// The bucket needs to be enqueued since it was an expired bucket.
			// We only need to enqueue the bucket when its expiration time has changed,
			// i.e. the wheel has advanced and this bucket get reused with a new expiration.
			// Any further calls to set the expiration within the same wheel cycle will
			// pass in the same value and hence return false, thus the bucket with the
			// same expiration will not be enqueued multiple times.
			// 将该bucket加入到延迟队列中
			tw.queue.Offer(b, b.Expiration())
		}

		return true
	} else {
		// Out of the interval. Put it into the overflow wheel
		// 如果放入的到期时间超过第一层时间轮，那么放到上一层中去
		overflowWheel := atomic.LoadPointer(&tw.overflowWheel)
		if overflowWheel == nil {
			atomic.CompareAndSwapPointer(
				&tw.overflowWheel,
				nil,
				// 需要注意的是，这里tick变成了interval
				unsafe.Pointer(newTimingWheel(
					tw.interval,
					tw.wheelSize,
					currentTime,
					tw.queue,
				)),
			)
			overflowWheel = atomic.LoadPointer(&tw.overflowWheel)
		}
		// 往上递归
		return (*TimingWheel)(overflowWheel).add(t)
	}
}

// addOrRun inserts the timer t into the current timing wheel, or run the
// timer's task if it has already expired.
func (tw *TimingWheel) addOrRun(t *Timer) {
	// 如果已经过期，那么直接执行
	if !tw.add(t) {
		// Already expired

		// Like the standard time.AfterFunc (https://golang.org/pkg/time/#AfterFunc),
		// always execute the timer's task in its own goroutine.
		// 异步执行定时任务
		go t.task()
	}
}

func (tw *TimingWheel) advanceClock(expiration int64) {
	currentTime := atomic.LoadInt64(&tw.currentTime)
	// 过期时间大于等于（当前时间+tick）
	if expiration >= currentTime+tw.tick {
		// 将currentTime设置为expiration，从而推进currentTime
		currentTime = truncate(expiration, tw.tick)
		atomic.StoreInt64(&tw.currentTime, currentTime)

		// Try to advance the clock of the overflow wheel if present
		// 如果有上层时间轮，那么递归调用上层时间轮的引用
		overflowWheel := atomic.LoadPointer(&tw.overflowWheel)
		if overflowWheel != nil {
			(*TimingWheel)(overflowWheel).advanceClock(currentTime)
		}
	}
}

// Start starts the current timing wheel.
func (tw *TimingWheel) Start() {
	// Poll会执行一个无限循环，将到期的元素放入到queue的C管道中
	tw.waitGroup.Wrap(func() {
		tw.queue.Poll(tw.exitC, func() int64 {
			return timeToMs(time.Now().UTC())
		})
	})
	// 开启无限循环获取queue中C的数据
	tw.waitGroup.Wrap(func() {
		for {
			select {
			// 从队列里面出来的数据都是到期的bucket
			case elem := <-tw.queue.C:
				b := elem.(*bucket)
				// 时间轮会将当前时间 currentTime 往前移动到 bucket的到期时间
				tw.advanceClock(b.Expiration())
				// 取出bucket队列的数据，并调用addOrRun方法执行
				b.Flush(tw.addOrRun)
			case <-tw.exitC:
				return
			}
		}
	})
}

// Stop stops the current timing wheel.
//
// If there is any timer's task being running in its own goroutine, Stop does
// not wait for the task to complete before returning. If the caller needs to
// know whether the task is completed, it must coordinate with the task explicitly.
func (tw *TimingWheel) Stop() {
	close(tw.exitC)
	tw.waitGroup.Wait()
}

// AfterFunc waits for the duration to elapse and then calls f in its own goroutine.
// It returns a Timer that can be used to cancel the call using its Stop method.
func (tw *TimingWheel) AfterFunc(d time.Duration, f func()) *Timer {
	t := &Timer{
		expiration: timeToMs(time.Now().UTC().Add(d)),
		task:       f,
	}
	tw.addOrRun(t)
	return t
}

// Scheduler determines the execution plan of a task.
type Scheduler interface {
	// Next returns the next execution time after the given (previous) time.
	// It will return a zero time if no next time is scheduled.
	//
	// All times must be UTC.
	Next(time.Time) time.Time
}

// ScheduleFunc calls f (in its own goroutine) according to the execution
// plan scheduled by s. It returns a Timer that can be used to cancel the
// call using its Stop method.
//
// If the caller want to terminate the execution plan halfway, it must
// stop the timer and ensure that the timer is stopped actually, since in
// the current implementation, there is a gap between the expiring and the
// restarting of the timer. The wait time for ensuring is short since the
// gap is very small.
//
// Internally, ScheduleFunc will ask the first execution time (by calling
// s.Next()) initially, and create a timer if the execution time is non-zero.
// Afterwards, it will ask the next execution time each time f is about to
// be executed, and f will be called at the next execution time if the time
// is non-zero.
func (tw *TimingWheel) ScheduleFunc(s Scheduler, f func()) (t *Timer) {
	expiration := s.Next(time.Now().UTC())
	if expiration.IsZero() {
		// No time is scheduled, return nil.
		return
	}

	t = &Timer{
		expiration: timeToMs(expiration),
		task: func() {
			// Schedule the task to execute at the next time if possible.
			expiration := s.Next(msToTime(t.expiration))
			if !expiration.IsZero() {
				t.expiration = timeToMs(expiration)
				tw.addOrRun(t)
			}

			// Actually execute the task.
			f()
		},
	}
	tw.addOrRun(t)

	return
}
