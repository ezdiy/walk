package walk

import (
	"runtime"
	"sync"
)

var Threaded bool
var MsgLoopMutex sync.Mutex

func EnterThread() {
	if !Threaded {
		panic("call me only in threaded mode")
	}
	runtime.LockOSThread()
	LockThread()
}

func LeaveThread() {
	if !Threaded {
		panic("call me only in threaded mode")
	}
	UnlockThread()
	runtime.UnlockOSThread()
}

func LockThread() {
	if Threaded {
		MsgLoopMutex.Lock()
	}
}

func UnlockThread() {
	if Threaded {
		MsgLoopMutex.Unlock()
	}
}

func RunUnlocked(f func()) {
	UnlockThread()
	defer LockThread()
	f()
}
