#ifndef SLIMY_THREADSHIM_H
#define SLIMY_THREADSHIM_H

#ifdef _WIN32
// threads.h shim for Windows

#include <windows.h>

enum {
	thrd_success,
	thrd_error,
};

typedef struct {HANDLE h;} thrd_t;

static inline int thrd_create(thrd_t *thr, LPTHREAD_START_ROUTINE func, void *arg) {
	thr->h = CreateThread(NULL, 0, func, arg, 0, NULL);
	if (!thr->h) return thrd_error;
	return thrd_success;
}

static inline int thrd_join(thrd_t thr, int *res) {
	if (WaitForSingleObject(thr.h, INFINITE) != WAIT_OBJECT_0) return thrd_error;
	if (res) {
		DWORD dwres;
		if (!GetExitCodeThread(thr.h, &dwres)) return thrd_error;
		*res = dwres;
	}
	return thrd_success;
}

static inline void thrd_close(thrd_t thr) {
	CloseHandle(thr.h);
}

#define THREAD_RET DWORD WINAPI

#else

#include <threads.h>
#define THREAD_RET int
static inline void thrd_close(thrd_t thr) {}

#endif

#endif
