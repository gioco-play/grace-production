package pchelper

import (
	"fmt"
	"github.com/smallnest/chanx"
	"golang.org/x/net/context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Set struct {
	Wg        sync.WaitGroup
	Signal    chan os.Signal
	Ctx       context.Context
	CtxCancel context.CancelFunc
	Close     bool
}

func New() *Set {
	ctx, cancel := context.WithCancel(context.Background())
	return &Set{
		Signal:    make(chan os.Signal, 1),
		Ctx:       ctx,
		CtxCancel: cancel,
		Close:     false,
	}
}

func (s *Set) Add(i int) {
	s.Wg.Add(i)
}

func (s *Set) Done() {
	s.Wg.Done()
}

func (s *Set) Wait() {
	s.Wg.Wait()
}

func (s *Set) Background() {
	s.Add(1)
	signal.Notify(s.Signal, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM,
		syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)
sign:
	for f := range s.Signal {
		switch f {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			if !s.Close {
				s.Close = true
				s.CtxCancel()
				break sign
			}
		default:

		}
	}

	defer func() {
		fmt.Println("信號關閉")
		s.Done()
	}()
}

type ProduceFunc func(Ch, ...interface{})

func (s *Set) Produce(f ProduceFunc, c Ch, params ...interface{}) {
	s.Add(1)
	f(c, params...)
	defer func() {
		fmt.Println("生產者關閉")
		s.Done()
	}()
}

type ComsumeFunc func(interface{}, ...interface{})

func (s *Set) Comsume(f ComsumeFunc, c Ch, params ...interface{}) {
	s.Add(1)
	flag := false
loop:
	for {
		select {
		case v, ok := <-c.Get():
			if ok {
				f(v, params...)

			} else {
				flag = true
				break loop
			}
		case <-s.Ctx.Done():
			if flag {
				break loop
			}

		}

		time.Sleep(1 * time.Second)
	}

	defer func() {
		fmt.Println("消費者關閉")
		s.Done()
	}()
}

type Ch interface {
	Get() <-chan chanx.T
	Set(interface{})
	Len() int
	Close()
}

type Chx struct {
	Ch *chanx.UnboundedChan
}

func NewChx(u *chanx.UnboundedChan) *Chx {
	return &Chx{
		Ch: u,
	}
}

func (c *Chx) Get() <-chan chanx.T {
	return c.Ch.Out
}

func (c *Chx) Set(i interface{}) {
	c.Ch.In <- i
}

func (c *Chx) Len() int {
	return c.Ch.Len()
}

func (c *Chx) Close() {
	close(c.Ch.In)
}
