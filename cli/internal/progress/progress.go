package progress

import (
	"io"
	"sync"

	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

// Tracker is safe to update from concurrent download workers.
type Tracker struct {
	bar  *mpb.Bar
	once sync.Once
	pool *mpb.Progress
}

func New(total int64, label string, output io.Writer) *Tracker {
	pool := mpb.New(mpb.WithWidth(58), mpb.WithOutput(output))
	bar := pool.AddBar(total,
		mpb.PrependDecorators(decor.Name(label+" ")),
		mpb.AppendDecorators(decor.CountersKibiByte("% .1f / % .1f"), decor.Percentage()),
	)
	return &Tracker{bar: bar, pool: pool}
}

func (t *Tracker) Add(n int) {
	if n > 0 {
		t.bar.IncrBy(n)
	}
}

func (t *Tracker) Complete() {
	t.once.Do(func() {
		t.bar.SetTotal(t.bar.Current(), true)
		t.pool.Wait()
	})
}
