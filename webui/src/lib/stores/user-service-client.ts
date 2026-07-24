import { createGrpcWebTransport } from '@connectrpc/connect-web';
import { createClient, ConnectError, Code, type Interceptor, type StreamResponse, type UnaryResponse } from '@connectrpc/connect';
import { UserService } from '$lib/api/v1/clients/user_service_pb';
import { getAccessToken, doRefresh, sessionExpired } from './auth-store';

// Wraps a streaming response's message iterator so that an Unauthenticated
// error raised while iterating (not just at call setup) is also observed and
// handled the same way as one raised at setup.
async function* observeUnauthenticated<T>(messages: AsyncIterable<T>): AsyncIterable<T> {
	try {
		for await (const message of messages) {
			yield message;
		}
	} catch (err) {
		await handleUnauthenticated(err);
		throw err;
	}
}

async function handleUnauthenticated(err: unknown) {
	if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
		const refreshed = await doRefresh();
		if (!refreshed) {
			sessionExpired();
		}
	}
}

// authInterceptor injects the current access token into every call, and
// reacts to Unauthenticated errors (the server revokes access tokens
// immediately on logout-elsewhere, unpair, or admin revocation):
//  - Unary calls: attempt one token refresh and retry once with the fresh
//    token. If the refresh fails the session is truly gone - clear auth
//    state and rethrow so the caller sees the original error.
//  - Streaming calls: request iterators can't be safely replayed, so we
//    never retry here. Instead we attempt a refresh so the *next* dial has a
//    fresh token, and clear auth state if that refresh fails. The existing
//    Streamer reconnect loop takes care of redialing (or, once auth state is
//    cleared, the route guard takes the user to /login and the streamer is
//    torn down).
const authInterceptor: Interceptor = (next) => async (req) => {
	req.header.set('authorization', getAccessToken());

	if (!req.stream) {
		try {
			return await next(req);
		} catch (err) {
			if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
				const refreshed = await doRefresh();
				if (refreshed) {
					req.header.set('authorization', getAccessToken());
					return (await next(req)) as UnaryResponse;
				}
				sessionExpired();
			}
			throw err;
		}
	}

	let res: StreamResponse;
	try {
		res = (await next(req)) as StreamResponse;
	} catch (err) {
		await handleUnauthenticated(err);
		throw err;
	}

	return {
		...res,
		message: observeUnauthenticated(res.message)
	};
};

// Create the GRPC-Web transport and client.
const transport = createGrpcWebTransport({
	baseUrl: '/api',
	interceptors: [authInterceptor]
});
const client = createClient(UserService, transport);

export {
	client as UserServiceClient
}
