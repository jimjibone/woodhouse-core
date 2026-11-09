import { Service_ServiceType } from '$lib/api/v2/clients/client_service_pb';
import { SERVICES } from '$lib/schema/zz_services';
import { endpointLabel, findEndpoint, isRootEndpoint } from '$lib/device';
import type { DevicesStoreDevice } from '$lib/stores/devices-stream';
import { memberKey } from '$lib/stores/groups-stream';

// Labels that the schema name does not give by itself.
const labelOverrides: Partial<Record<Service_ServiceType, string>> = {
	[Service_ServiceType.ON_OFF]: 'On/off'
};

// Every schema service type, in schema order, so new types show up without
// changes here. "device_temperature" reads as "Device temperature".
export const serviceTypeOptions: { value: Service_ServiceType; label: string }[] = SERVICES.map((srv) => ({
	value: srv.type,
	label: labelOverrides[srv.type] ?? srv.name.charAt(0).toUpperCase() + srv.name.slice(1).replaceAll('_', ' ')
}));

export const typeLabel = (type: Service_ServiceType): string =>
	serviceTypeOptions.find((o) => o.value === type)?.label ?? 'Unknown';

type MemberRef = { deviceId: string; endpointId: string; serviceId: string };

/**
 * "Device name / Endpoint label / service". The endpoint part is omitted for
 * the root endpoint. The service part is its alias, else its ID. Falls back
 * to IDs for anything the devices store does not (yet) know.
 */
export const memberLabel = (devices: readonly DevicesStoreDevice[], m: MemberRef): string => {
	const device = devices.find((d) => d.id === m.deviceId);
	const parts = [device?.name ?? m.deviceId];
	if (!isRootEndpoint(m.endpointId)) {
		const ep = device ? findEndpoint(device, m.endpointId) : undefined;
		parts.push(ep ? endpointLabel(ep) : `Endpoint ${m.endpointId}`);
	}
	const service = device
		?.endpoints.find((e) => e.id === m.endpointId)
		?.services.find((s) => s.id === m.serviceId);
	parts.push(service?.alias || m.serviceId);
	return parts.join(' / ');
};

export const keyOf = (m: MemberRef): string => memberKey(m.deviceId, m.endpointId, m.serviceId);
