// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package runtime

import (
	"runtime/internal/atomic"
	"runtime/internal/sys"
	"unsafe"
)

const (
	_AT_NULL     = 0
	_AT_PLATFORM = 15 //  introduced in at least 2.6.11
	_AT_HWCAP    = 16 // introduced in at least 2.6.11
	_AT_RANDOM   = 25 // introduced in 2.6.29

	_HWCAP_VFP   = 1 << 6  // introduced in at least 2.6.11
	_HWCAP_VFPv3 = 1 << 13 // introduced in 2.6.30
)

var randomNumber uint32
var armArch uint8 = 6 // we default to ARMv6
var hwcap uint32      // set by setup_auxv

func checkgoarm() {
	//printstring("checkgoarm\n")
	if goarm > 5 && hwcap&_HWCAP_VFP == 0 {
		print("runtime: this CPU has no floating point hardware, so it cannot run\n")
		print("this GOARM=", goarm, " binary. Recompile using GOARM=5.\n")
		exit(1)
	}
	if goarm > 6 && hwcap&_HWCAP_VFPv3 == 0 {
		print("runtime: this CPU has no VFPv3 floating point hardware, so it cannot run\n")
		print("this GOARM=", goarm, " binary. Recompile using GOARM=5.\n")
		exit(1)
	}
}

func sysargs(argc int32, argv **byte) {
	printstring("sysargs\n")
	armArch = 7
	hwcap = 0xFFFFFFFF
	return
	// skip over argv, envv to get to auxv
	n := argc + 1
	for argv_index(argv, n) != nil {
		n++
	}
	n++
	auxv := (*[1 << 28]uint32)(add(unsafe.Pointer(argv), uintptr(n)*sys.PtrSize))

	for i := 0; auxv[i] != _AT_NULL; i += 2 {
		switch auxv[i] {
		case _AT_RANDOM: // kernel provides a pointer to 16-bytes worth of random data
			startupRandomData = (*[16]byte)(unsafe.Pointer(uintptr(auxv[i+1])))[:]
			// the pointer provided may not be word aligned, so we must treat it
			// as a byte array.
			randomNumber = uint32(startupRandomData[4]) | uint32(startupRandomData[5])<<8 |
				uint32(startupRandomData[6])<<16 | uint32(startupRandomData[7])<<24

		case _AT_PLATFORM: // v5l, v6l, v7l
			t := *(*uint8)(unsafe.Pointer(uintptr(auxv[i+1] + 1)))
			if '5' <= t && t <= '7' {
				armArch = t - '0'
			}

		case _AT_HWCAP: // CPU capability bit flags
			hwcap = auxv[i+1]
		}
	}
}

//go:nosplit
func cputicks() int64 {
	// Currently cputicks() is used in blocking profiler and to seed fastrand1().
	// nanotime() is a poor approximation of CPU ticks that is enough for the profiler.
	// randomNumber provides better seeding of fastrand1.
	return nanotime() + int64(randomNumber)
}

//biscuit armv7a mods
func SWIcall()
func Runtime_main()

type Spinlock_t struct {
	v uint32
}

//go:nosplit
func Splock(l *Spinlock_t) {
	for {
		if atomic.Xchg(&l.v, 1) == 1 {
			break
		}
		//for l.v != 0 {
		//htpause()
		//}
	}
}

//go:nosplit
func Spunlock(l *Spinlock_t) {
	//atomic.Store(&l.v, 0)
	l.v = 0
}

func mktrap(a int) {
	print("mktrap")
}

func trapcheck(pp *p) {
	print("trapcheck")
}

func printtest() {
	throw("help")
}

func cascheck() {
	var z uint32
	z = 1
	if !atomic.Cas(&z, 1, 2) {
		throw("cascheck 1")
	}
	if z != 2 {
		throw("cascheck z not 2")
	}

	z = 3
	if !atomic.Cas(&z, 3, 4) {
		throw("cascheck 2")
	}
	writeUnsafe([]byte("cascheck pass\n"))
	//print("kernel start: ", hex(kernelstart), " kernel size: ", hex(kernelsize), "\n")
}

//func hack_mmap(va, _sz uintptr, _prot uint32, _flags uint32,
//	fd int32, offset int32) uintptr {
//	//	fl := Pushcli()
//	Splock(maplock)
//	print("hack_mmap: ", hex(va), " ", hex(_sz), " ", hex(_prot), " ", hex(_flags), "\n")
//	Spunlock(maplock)
//	return 0
//	//	MAP_ANON := uintptr(0x20)
//	//	MAP_PRIVATE := uintptr(0x2)
//	//	PROT_NONE := uintptr(0x0)
//	//	PROT_WRITE := uintptr(0x2)
//	//
//	//	prot := uintptr(_prot)
//	//	flags := uintptr(_flags)
//	//	var vaend uintptr
//	//	var perms uintptr
//	//	var ret uintptr
//	//	var t uintptr
//	//	pgleft := pglast - pgfirst
//	//	sz := pgroundup(_sz)
//	//	if sz > pgleft {
//	//		ret = ^uintptr(0)
//	//		goto out
//	//	}
//	//	sz = pgroundup(va + _sz)
//	//	sz -= pgrounddown(va)
//	//	if va == 0 {
//	//		va = find_empty(sz)
//	//	}
//	//	vaend = caddr(VUEND, 0, 0, 0, 0)
//	//	if va >= vaend || va+sz >= vaend {
//	//		pancake("va space exhausted", va)
//	//	}
//	//
//	//	t = MAP_ANON | MAP_PRIVATE
//	//	if flags&t != t {
//	//		pancake("unexpected flags", flags)
//	//	}
//	//	perms = PTE_P
//	//	if prot == PROT_NONE {
//	//		prot_none(va, sz)
//	//		ret = va
//	//		goto out
//	//	}
//	//
//	//	if prot&PROT_WRITE != 0 {
//	//		perms |= PTE_W
//	//	}
//	//
//	//	if _nopml4 {
//	//		eidx := pml4x(va + sz - 1)
//	//		for sidx := pml4x(va); sidx <= eidx; sidx++ {
//	//			pml4 := caddr(VREC, VREC, VREC, VREC, sidx)
//	//			pml4e := (*uintptr)(unsafe.Pointer(pml4))
//	//			if *pml4e&PTE_P == 0 {
//	//				pancake("new pml4 entry to kernel pmap", va)
//	//			}
//	//		}
//	//	}
//	//
//	//	for i := uintptr(0); i < sz; i += PGSIZE {
//	//		alloc_map(va+i, perms, true)
//	//	}
//	//	ret = va
//	//out:
//	//	Spunlock(maplock)
//	//	Popcli(fl)
//	//	return ret
//}
