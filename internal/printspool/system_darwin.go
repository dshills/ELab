//go:build darwin && cgo

package printspool

/*
#cgo LDFLAGS: -lcups
#include <cups/cups.h>
#include <sys/socket.h>
#include <stdlib.h>
#include <stdio.h>

static const char *elabNoPassword(const char *prompt, http_t *http, const char *method, const char *resource, void *data) { return NULL; }
static http_t *elabPrintConnect(const char *socket) {
  cupsSetPasswordCB2(elabNoPassword, NULL);
  http_t *h = httpConnect2(socket, 631, NULL, AF_UNSPEC, HTTP_ENCRYPTION_IF_REQUESTED, 1, 2000, NULL);
  if (h) httpSetTimeout(h, 5.0, NULL, NULL);
  return h;
}
static const char *elabDestOption(cups_dest_t *d, const char *name) {
  const char *v = cupsGetOption(name, d->num_options, d->options);
  return v ? v : "";
}
static int elabSubmit(http_t *h, const char *printer, const char *path, const char *title, const char *paper, const char *orientation, const char *copies) {
  cups_option_t *options = NULL;
  int n = 0;
  n = cupsAddOption("document-format", "application/pdf", n, &options);
  n = cupsAddOption("media", paper, n, &options);
  n = cupsAddOption("orientation-requested", orientation, n, &options);
  n = cupsAddOption("copies", copies, n, &options);
  n = cupsAddOption("print-scaling", "none", n, &options);
  n = cupsAddOption("sides", "one-sided", n, &options);
  n = cupsAddOption("job-sheets", "none,none", n, &options);
  int id = cupsPrintFile2(h, printer, path, title, n, options);
  cupsFreeOptions(n, options);
  return id;
}
static int elabJobState(http_t *h, int id) {
  ipp_t *request = ippNewRequest(IPP_OP_GET_JOB_ATTRIBUTES);
  char uri[128];
  snprintf(uri, sizeof(uri), "ipp://localhost/jobs/%d", id);
  ippAddString(request, IPP_TAG_OPERATION, IPP_TAG_URI, "job-uri", NULL, uri);
  ippAddString(request, IPP_TAG_OPERATION, IPP_TAG_NAME, "requesting-user-name", NULL, cupsUser());
  ippAddString(request, IPP_TAG_OPERATION, IPP_TAG_KEYWORD, "requested-attributes", NULL, "job-state");
  ipp_t *response = cupsDoRequest(h, request, "/jobs/");
  if (!response) return 0;
  ipp_attribute_t *a = ippFindAttribute(response, "job-state", IPP_TAG_ENUM);
  int state = a ? ippGetInteger(a, 0) : 0;
  ippDelete(response);
  return state;
}
*/
import "C"

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"unsafe"
)

// CUPS error/auth state is thread-local. Each call owns a connection and stays
// on one OS thread through error capture and cleanup. No interactive auth prompt.
func connect(ctx context.Context, socket string) (*C.http_t, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if socket == "" {
		socket = "/private/var/run/cupsd"
	}
	address, free := cstring(socket)
	defer free()
	h := C.elabPrintConnect(address)
	if h == nil {
		return nil, fmt.Errorf("cannot connect to local print service")
	}
	return h, nil
}
func cupsError() error {
	return fmt.Errorf("local print service: %s", C.GoString(C.cupsLastErrorString()))
}
func cstring(s string) (*C.char, func()) {
	p := C.CString(s)
	return p, func() { C.free(unsafe.Pointer(p)) }
}
func (b systemBackend) Destinations(ctx context.Context) ([]Destination, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	h, err := connect(ctx, b.socket)
	if err != nil {
		return nil, err
	}
	defer C.httpClose(h)
	var list *C.cups_dest_t
	n := C.cupsGetDests2(h, &list)
	defer C.cupsFreeDests(n, list)
	if n < 0 || C.cupsLastError() >= C.IPP_STATUS_ERROR_BAD_REQUEST {
		return nil, cupsError()
	}
	if n > 128 {
		return nil, fmt.Errorf("local print service returned more than 128 destinations")
	}
	result := []Destination{}
	for _, d := range unsafe.Slice(list, int(n)) {
		if d.instance != nil {
			continue
		} // Named preference instances are not separate queues.
		option := func(name string) string {
			p, free := cstring(name)
			defer free()
			return C.GoString(C.elabDestOption(&d, p))
		}
		id, name := C.GoString(d.name), option("printer-info")
		if name == "" {
			name = id
		}
		state := map[string]string{"3": "idle", "4": "printing", "5": "stopped"}[option("printer-state")]
		if state == "" {
			state = "unknown"
		}
		result = append(result, Destination{ID: id, Name: name, Default: d.is_default != 0, Accepting: option("printer-is-accepting-jobs") == "true", State: state})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, ctx.Err()
}
func (b systemBackend) Submit(ctx context.Context, printer, path, title string, options Options) (Receipt, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	h, err := connect(ctx, b.socket)
	if err != nil {
		return Receipt{}, err
	}
	defer C.httpClose(h)
	if err := ctx.Err(); err != nil {
		return Receipt{}, err
	}
	p, fp := cstring(printer)
	defer fp()
	f, ff := cstring(path)
	defer ff()
	t, ft := cstring(title)
	defer ft()
	m, fm := cstring(options.Paper)
	defer fm()
	orientation := "3"
	if options.Orientation == "landscape" {
		orientation = "4"
	}
	o, fo := cstring(orientation)
	defer fo()
	c, fc := cstring(strconv.Itoa(options.Copies))
	defer fc()
	id := int(C.elabSubmit(h, p, f, t, m, o, c))
	if id > 0 {
		return Receipt{ID: id, State: "submitted"}, nil
	} // Preserve acknowledgment even after cancellation.
	code := C.cupsLastError()
	if code >= C.IPP_STATUS_ERROR_BAD_REQUEST && code < C.IPP_STATUS_ERROR_INTERNAL {
		return Receipt{}, cupsError()
	}
	return Receipt{}, fmt.Errorf("%w (%v)", ErrUncertain, cupsError())
}
func (b systemBackend) Inspect(ctx context.Context, id int) (Receipt, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	h, err := connect(ctx, b.socket)
	if err != nil {
		return Receipt{}, err
	}
	defer C.httpClose(h)
	state := int(C.elabJobState(h, C.int(id)))
	if C.cupsLastError() == C.IPP_STATUS_ERROR_NOT_FOUND {
		return Receipt{}, ErrNotFound
	}
	if C.cupsLastError() >= C.IPP_STATUS_ERROR_BAD_REQUEST {
		return Receipt{}, cupsError()
	}
	name := map[int]string{3: "pending", 4: "held", 5: "printing", 6: "stopped", 7: "canceled", 8: "aborted", 9: "completed"}[state]
	if name == "" {
		return Receipt{}, fmt.Errorf("local print service returned no known job state")
	}
	return Receipt{ID: id, State: name}, ctx.Err()
}
func (b systemBackend) Cancel(ctx context.Context, id int) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	h, err := connect(ctx, b.socket)
	if err != nil {
		return err
	}
	defer C.httpClose(h)
	if code := C.cupsCancelJob2(h, nil, C.int(id), 0); code >= C.IPP_STATUS_ERROR_BAD_REQUEST {
		return cupsError()
	}
	return nil // Acknowledgment only; refresh observes terminal state.
}
