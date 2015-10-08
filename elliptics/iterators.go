/*
 * 2013+ Copyright (c) Anton Tyurin <noxiouz@yandex.ru>
 * 2014+ Copyright (c) Evgeniy Polyakov <zbr@ioremap.net>
 * All rights reserved.
 *
 * This program is free software; you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation; either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU General Public License for more details.
 */

package elliptics

/*
#include "session.h"
#include <stdio.h>

struct dnet_iterator_response_unpacked {
        uint64_t                        id;
        struct dnet_raw_id              key;
        int                             status;
        struct dnet_time                timestamp;
        uint64_t                        user_flags;
        uint64_t                        size;
        uint64_t                        iterated_keys;
        uint64_t                        total_keys;
        uint64_t                        flags;
};

static inline void unpack_dnet_iterator_response(struct dnet_iterator_response *from,
	 struct dnet_iterator_response_unpacked *to)
{
	to->id = from->id;
    memcpy(to->key.id, from->key.id, DNET_ID_SIZE);
    to->status = from->status;
    to->timestamp = from->timestamp;
    to->user_flags = from->user_flags;
    to->size = from->size;
    to->iterated_keys = from->iterated_keys;
    to->total_keys = from->total_keys;
    to->flags = from->flags;
}

*/
import "C"

import (
	"fmt"
	"time"
	"unsafe"
)

type DnetIteratorResponse struct {
	Id           uint64
	Key          C.struct_dnet_raw_id
	Status       int
	Timestamp    time.Time
	UserFlags    uint64
	Size         uint64
	IteratedKeys uint64
	TotalKeys    uint64
	Flags        uint64
}

type IteratorResult interface {
	Reply() *DnetIteratorResponse
	ReplyData() []byte
	Id() uint64
	Error() error
}

type iteratorResult struct {
	reply     *DnetIteratorResponse
	replyData []byte
	id        uint64
	err       error
}

type dnetRawId [C.DNET_ID_SIZE]uint8

type DnetIteratorRange struct {
	KeyBegin dnetRawId
	KeyEnd   dnetRawId
}

func (i *iteratorResult) Reply() *DnetIteratorResponse { return i.reply }

func (i *iteratorResult) ReplyData() []byte { return i.replyData }

func (i *iteratorResult) Id() uint64 { return i.id }

func (i *iteratorResult) Error() error { return i.err }

//export go_iterator_callback
func go_iterator_callback(result *C.struct_go_iterator_result, key uint64) {
	context, err := Pool.Get(key)
	if err != nil {
		panic("Unable to find session number")
	}

	callback := context.(func(*iteratorResult))

	var reply C.struct_dnet_iterator_response_unpacked

	C.unpack_dnet_iterator_response(result.reply, &reply)

	var Result = iteratorResult{
		reply: &DnetIteratorResponse{
			Id:           uint64(result.reply.id),
			Key:          reply.key,
			Status:       int(reply.status),
			Timestamp:    time.Unix(int64(reply.timestamp.tsec), int64(reply.timestamp.tnsec)),
			UserFlags:    uint64(reply.user_flags),
			Size:         uint64(reply.size),
			IteratedKeys: uint64(reply.iterated_keys),
			TotalKeys:    uint64(reply.total_keys),
			Flags:        uint64(reply.flags),
		},
		id:        uint64(result.id),
		replyData: nil,
	}

	if result.reply_size > 0 && result.reply_data != nil {
		Result.replyData = C.GoBytes(unsafe.Pointer(result.reply_data), C.int(result.reply_size))
	} else {
		Result.replyData = make([]byte, 0)
	}

	callback(&Result)
}

func iteratorHelper(key string) (*Key, uint64, uint64, <-chan IteratorResult) {
	ekey, err := NewKey(key)
	if err != nil {
		panic(err)
	}

	responseCh := make(chan IteratorResult, defaultVOLUME)

	onResultContext := NextContext()
	onFinishContext := NextContext()

	onResult := func(iterres *iteratorResult) {
		responseCh <- iterres
	}

	onFinish := func(err error) {
		if err != nil {
			responseCh <- &iteratorResult{err: err}
		}
		close(responseCh)

		Pool.Delete(onResultContext)
		Pool.Delete(onFinishContext)
	}

	Pool.Store(onResultContext, onResult)
	Pool.Store(onFinishContext, onFinish)
	return ekey, onResultContext, onFinishContext, responseCh
}

func adjustTimeFrame(ctime_begin, ctime_end *C.struct_dnet_time, timeFrame ...time.Time) error {
	switch count := len(timeFrame); {
	case count >= 1: // set time begin
		if !timeFrame[0].IsZero() {
			ctime_begin.tnsec = C.uint64_t(timeFrame[0].UnixNano())
		}
		fallthrough
	case count == 2: // set both
		if !timeFrame[1].IsZero() {
			ctime_end.tnsec = C.uint64_t(timeFrame[1].UnixNano())
		}
	default:
		return fmt.Errorf("no more than 2 items can be passed as timeFrame")
	}

	return fmt.Errorf("no less than 1 item can be passed as timeFrame")
}

func (s *Session) IteratorStart(key string, ranges []DnetIteratorRange, Type uint64, flags uint64, timeFrame ...time.Time) <-chan IteratorResult {
	ekey, onResultContext, onFinishContext, responseCh := iteratorHelper(key)
	defer ekey.Free()

	var ctime_begin, ctime_end C.struct_dnet_time

	if err := adjustTimeFrame(&ctime_begin, &ctime_end, timeFrame...); err != nil {
		context, pool_err := Pool.Get(onFinishContext)
		if pool_err != nil {
			panic("Unable to find session number")
		}
		context.(func(error))(err)
		return responseCh
	}

	var cranges = make([]C.struct_go_iterator_range, 0, len(ranges))
	// Seems it's redundant copying
	for _, rng := range ranges {
		cranges = append(cranges, C.struct_go_iterator_range{
			(*C.uint8_t)(&rng.KeyBegin[0]),
			(*C.uint8_t)(&rng.KeyEnd[0]),
		})
	}

	C.session_start_iterator(s.session, C.context_t(onResultContext), C.context_t(onFinishContext),
		(*C.struct_go_iterator_range)(&cranges[0]),
		C.size_t(len(ranges)),
		ekey.key,
		C.uint64_t(Type),
		C.uint64_t(flags),
		ctime_begin,
		ctime_end)
	return responseCh
}

func (s *Session) IteratorPause(key string, iteratorId uint64) <-chan IteratorResult {
	ekey, onResultContext, onFinishContext, responseCh := iteratorHelper(key)
	defer ekey.Free()

	C.session_pause_iterator(s.session, C.context_t(onResultContext), C.context_t(onFinishContext),
		ekey.key,
		C.uint64_t(iteratorId))
	return responseCh
}

func (s *Session) IteratorContinue(key string, iteratorId uint64) <-chan IteratorResult {
	ekey, onResultContext, onFinishContext, responseCh := iteratorHelper(key)
	defer ekey.Free()

	C.session_continue_iterator(s.session, C.context_t(onResultContext), C.context_t(onFinishContext),
		ekey.key,
		C.uint64_t(iteratorId))
	return responseCh
}

func (s *Session) IteratorCancel(key string, iteratorId uint64) <-chan IteratorResult {
	ekey, onResultContext, onFinishContext, responseCh := iteratorHelper(key)
	defer ekey.Free()

	C.session_cancel_iterator(s.session, C.context_t(onResultContext), C.context_t(onFinishContext),
		ekey.key,
		C.uint64_t(iteratorId))
	return responseCh
}

func (s *Session) CopyIteratorStart(key string, ranges []DnetIteratorRange, groups []uint32, flags uint64, timeFrame ...time.Time) <-chan IteratorResult {
	ekey, onResultContext, onFinishContext, responseCh := iteratorHelper(key)
	defer ekey.Free()

	var ctime_begin, ctime_end C.struct_dnet_time

	if err := adjustTimeFrame(&ctime_begin, &ctime_end, timeFrame...); err != nil {
		context, pool_err := Pool.Get(onFinishContext)
		if pool_err != nil {
			panic("Unable to find session number")
		}
		context.(func(error))(err)
		return responseCh
	}

	var cranges = make([]C.struct_go_iterator_range, 0, len(ranges))
	// Seems it's redundant copying
	for _, rng := range ranges {
		cranges = append(cranges, C.struct_go_iterator_range{
			(*C.uint8_t)(&rng.KeyBegin[0]),
			(*C.uint8_t)(&rng.KeyEnd[0]),
		})
	}

	C.session_start_copy_iterator(s.session, C.context_t(onResultContext), C.context_t(onFinishContext),
		(*C.struct_go_iterator_range)(&cranges[0]), C.size_t(len(ranges)),
		(*C.uint32_t)(&groups[0]), (C.size_t)(len(groups)),
		ekey.key,
		C.uint64_t(flags),
		ctime_begin,
		ctime_end)

	return responseCh
}

func (s *Session) ServerSend(keys []string, flags uint64, groups []uint32) <-chan IteratorResult {
	responseCh := make(chan IteratorResult, defaultVOLUME)

	kkeys, err := NewKeys(keys)
	if err != nil {
		responseCh <- &iteratorResult{err: err}
		close(responseCh)
		return responseCh
	}
	defer kkeys.Free()

	onResultContext := NextContext()
	onFinishContext := NextContext()

	onResult := func(iterres *iteratorResult) {
		responseCh <- iterres
	}

	onFinish := func(err error) {
		if err != nil {
			responseCh <- &iteratorResult{err: err}
		}
		close(responseCh)

		Pool.Delete(onResultContext)
		Pool.Delete(onFinishContext)
	}

	Pool.Store(onResultContext, onResult)
	Pool.Store(onFinishContext, onFinish)

	C.session_server_send(s.session, C.context_t(onResultContext), C.context_t(onFinishContext),
		kkeys.keys,
		C.uint64_t(flags),
		(*C.uint32_t)(&groups[0]), (C.size_t)(len(groups)))

	return responseCh
}
