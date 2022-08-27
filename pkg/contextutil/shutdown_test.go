package contextutil

import (
	"context"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

var locker sync.Mutex

func TestWithShutdownSigTerm(t *testing.T) {
	locker.Lock()
	defer locker.Unlock()
	testDone := make(chan interface{})
	ctx := WithShutdown(context.Background(), func(sig os.Signal) {
		defer func() {
			testDone <- nil
		}()
		if sig == nil {
			t.Errorf("signal should not be nil")
			return
		}
		if sig.String() != syscall.SIGTERM.String() {
			t.Errorf("received invalid stop signal %s", sig.String())
			return
		}
		t.Logf("received stop signal %s", sig.String())
	})
	go func(pid int) {
		p, err := process.NewProcess(int32(pid))
		if err != nil {
			t.Errorf("process pid %d does not exist", pid)
			return
		}
		err = p.Terminate()
		if err != nil {
			t.Errorf("error sending SIGTERM err = %v", err)
			return
		}
	}(syscall.Getpid())
	testTimeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case <-testTimeoutCtx.Done():
		t.Errorf("test timed out")
		return
	case <-ctx.Done():
		// context successfully canceled
	}
	// prevent log from not firing after terminated
	<-testDone
}

func TestWithShutdownParentCanceled(t *testing.T) {
	locker.Lock()
	defer locker.Unlock()
	testDone := make(chan interface{})
	super, superCancel := context.WithCancel(context.Background())
	ctx := WithShutdown(super, func(sig os.Signal) {
		defer func() {
			testDone <- nil
		}()
		if sig != nil {
			t.Errorf("signal should be nil but received: %s", sig.String())
			return
		}
		t.Logf("received nil signal successfully")
	})
	superCancel()
	testTimeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	select {
	case <-testTimeoutCtx.Done():
		t.Errorf("test timed out")
		return
	case <-ctx.Done():
		// context successfully canceled
	}
	// prevent log from not firing after terminated
	<-testDone
}
