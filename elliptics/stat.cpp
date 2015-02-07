/*
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

#include "session.h"
#include "stat.h"

using namespace ioremap;

extern "C" {

//This header is generated by cgo at compile time
#include "_cgo_export.h"

void on_finish(context_t context, const elliptics::error_info &error);

static void on_stat_one(context_t context, const elliptics::monitor_stat_result_entry &result)
{
	try {
		std::string stats = result.statistics();
		go_stat_result tmp {
			result.command(), result.address(), stats.c_str(), stats.size()
		};

		go_stat_callback(&tmp, context);
	} catch (const std::exception &e) {
		std::string res(e.what());

		go_stat_result tmp {
			result.command(), result.address(),
				res.c_str(), res.size()
		};

		go_stat_callback(&tmp, context);
	}
}

void session_get_stats(ell_session *session, context_t on_chunk_context, context_t final_context, uint64_t categories)
{
	session->monitor_stat(categories).connect(std::bind(&on_stat_one, on_chunk_context, std::placeholders::_1),
			std::bind(&on_finish, final_context, std::placeholders::_1));
}

} // extern "C"

