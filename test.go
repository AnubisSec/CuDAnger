package main

import (

	"fmt"
	"C"
	"strconv"
	"unsafe"
	"os/exec"


	"github.com/barnex/cuda5/cu"
)

func main() {
	// This must be called if this is the first time the API is being called
	// Must always be zero
	cu.Init(0);

	fmt.Println("Hello, I am you GPU:", cu.Device(0).Name())
	fmt.Println("Number of devices: " + strconv.Itoa(cu.DeviceGetCount()))
	fmt.Println("Free memory: " + strconv.FormatInt(cu.DeviceGet(0).TotalMem(),10))


/*
	Context is how the API wrapper communicates with the driver API,
	This is needed to be established in order to communicate from the CPU to the GPU
*/



	// Set current context
	ctx := cu.CtxCreate(cu.CTX_SCHED_AUTO, 0)
	cu.CtxSetCurrent(ctx)
	fmt.Println("Created CUDA context")


	// Allocate 2048 bytes in the GPU
	fmt.Println("[+] Attepmting to allocate memory...")
	Memptr := cu.MemAlloc(2048)
	fmt.Println("[+] Allocated memory pointer", Memptr)

	// MemCpy to device
	fmt.Println("[+] Writing data to newly allocated memory...")
	data := "whoami && id"
	cu.MemcpyHtoD(Memptr, unsafe.Pointer(&data), int64(len(data)))

	// Read data from the copy
	var data2 string;
	cu.MemcpyDtoH(unsafe.Pointer(&data2), Memptr, int64(len(data)))
	fmt.Println("[+] Reading data from", Memptr, "...")

	// Execute command that resides in GPU memeory
	fmt.Println("[+] Executing command from GPU memory...")
	out, _ := exec.Command("/bin/sh", "-c", data2).Output()
	fmt.Printf("%s", out)

	// Free the newly created mem
	fmt.Println("[+] Attempting to free allocated memory at", Memptr, "...")
	cu.MemFree(Memptr)
	fmt.Println("[+] Successfully free'd the allocated mem!")
}

