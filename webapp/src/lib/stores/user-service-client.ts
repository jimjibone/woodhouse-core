import { createGrpcWebTransport } from '@connectrpc/connect-web';
import { createPromiseClient } from '@connectrpc/connect';
import { UserService } from '$lib/api/v1/clients/user_service_connect';

// Create the GRPC-Web transport and client.
const transport = createGrpcWebTransport({
	baseUrl: '/api'
});
const client = createPromiseClient(UserService, transport);

export {
	client as UserServiceClient
}
