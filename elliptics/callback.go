package elliptics

/*
#include "session.h"
*/
import "C"

import (
	"fmt"
	"reflect"
	"unsafe"
)

var _ = fmt.Println

//export go_final_callback
func go_final_callback(cerr *C.struct_go_error, context unsafe.Pointer) {
	callback := *(*func(error))(context)

	if (cerr.code < 0) {
		err := &DnetError {
			Code:		int(cerr.code),
			Flags:		uint64(cerr.flags),
			Message:	C.GoString(cerr.message),
		}

		callback(err)
	} else {
		callback(nil)
	}
}

//export go_lookup_error
func go_lookup_error(cmd *C.struct_dnet_cmd, addr *C.struct_dnet_addr, cerr *C.struct_go_error, context unsafe.Pointer) {
	callback := *(*func(*lookupResult))(context)
	Result := lookupResult {
		cmd:	NewDnetCmd(cmd),
		addr:	NewDnetAddr(addr),
		err:	&DnetError {
				Code:		int(cerr.code),
				Flags:		uint64(cerr.flags),
				Message:	C.GoString(cerr.message),
			},
	}
	callback(&Result)
}

//export go_lookup_callback
func go_lookup_callback(result *C.struct_go_lookup_result, context unsafe.Pointer) {
	callback := *(*func(*lookupResult))(context)
	Result := lookupResult {
		cmd:	NewDnetCmd(result.cmd),
		addr:	NewDnetAddr(result.addr),
		info:	NewDnetFileInfo(result.info),
		storage_addr:	NewDnetAddr(result.storage_addr),
		path:	C.GoString(result.path),
		err:	nil,
	}
	callback(&Result)
}

//export go_remove_callback
func go_remove_callback(result *C.struct_go_remove_result, context unsafe.Pointer) {
	callback := *(*func(*removeResult))(context)
	Result := removeResult {
		cmd:	NewDnetCmd(result.cmd),
	}
	callback(&Result)
}

//export go_read_callback
func go_read_callback(result *C.struct_go_read_result, context unsafe.Pointer) {
	callback := *(*func(readResult))(context)

	Result := readResult {
		cmd:	NewDnetCmd(result.cmd),
		addr:	NewDnetAddr(result.addr),
		ioattr:	NewDnetIOAttr(result.io_attribute),
		data:	C.GoBytes(unsafe.Pointer(result.file), C.int(result.size)),
		err:	nil,
	}
	// All data from C++ has been copied here.
	callback(Result)
}

//export go_find_callback
func go_find_callback(result *C.struct_go_find_result, context unsafe.Pointer) {
	callback := *(*func(*findResult))(context)
	var indexEntries []C.struct_c_index_entry
	size := int(result.entries_count)
	sliceHeader := (*reflect.SliceHeader)((unsafe.Pointer(&indexEntries)))
	sliceHeader.Cap = size
	sliceHeader.Len = size
	sliceHeader.Data = uintptr(unsafe.Pointer(result.entries))
	var IndexDatas []IndexEntry
	for _, item := range indexEntries {
		IndexDatas = append(IndexDatas, IndexEntry{
			Data: C.GoStringN(item.data, C.int(item.size)),
		})
	}
	callback(&findResult{
		id:   *result.id,
		data: IndexDatas,
		err:  nil,
	})
}

//export go_index_entry_callback
func go_index_entry_callback(result *C.struct_c_index_entry, context unsafe.Pointer) {
	callback := *(*func(*IndexEntry))(context)
	callback(&IndexEntry{Data: C.GoStringN(result.data, C.int(result.size))})
}
