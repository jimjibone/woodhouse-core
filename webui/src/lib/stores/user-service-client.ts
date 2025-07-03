import { createGrpcWebTransport } from '@connectrpc/connect-web';
import { createClient } from '@connectrpc/connect';
import { UserService } from '$lib/api/v1/clients/user_service_pb';

// Create the GRPC-Web transport and client.
const transport = createGrpcWebTransport({
	baseUrl: '/api'
});
const client = createClient(UserService, transport);

export {
	client as UserServiceClient
}
