import { Service_ServiceType } from '$lib/api/v2/clients/client_service_pb';
import { endpointLabel, findEndpoint, isRootEndpoint } from '$lib/device';
import type { DevicesStoreDevice } from '$lib/stores/devices-stream';
import { memberKey } from '$lib/stores/groups-stream';

export const serviceTypeOptions: { value: Service_ServiceType; label: string }[] = [
	{ value: Service_ServiceType.INFO, label: 'Info' },
	{ value: Service_ServiceType.ONLINE, label: 'Online' },
	{ value: Service_ServiceType.IDENTIFY, label: 'Identify' },
	{ value: Service_ServiceType.UPDATE, label: 'Update' },
	{ value: Service_ServiceType.BATTERY, label: 'Battery' },
	{ value: Service_ServiceType.DEVICE_TEMPERATURE, label: 'Device temperature' },
	{ value: Service_ServiceType.ON_OFF, label: 'On/off' },
	{ value: Service_ServiceType.LIGHT, label: 'Light' },
	{ value: Service_ServiceType.COVER, label: 'Cover' },
	{ value: Service_ServiceType.THERMOSTAT, label: 'Thermostat' },
	{ value: Service_ServiceType.BINARY_INPUT, label: 'Binary input' },
	{ value: Service_ServiceType.ANALOG_INPUT, label: 'Analog input' },
	{ value: Service_ServiceType.BUTTON, label: 'Button' },
	{ value: Service_ServiceType.CONTACT, label: 'Contact' },
	{ value: Service_ServiceType.MOTION, label: 'Motion' },
	{ value: Service_ServiceType.PRESENCE, label: 'Presence' },
	{ value: Service_ServiceType.TEMPERATURE, label: 'Temperature' },
	{ value: Service_ServiceType.HUMIDITY, label: 'Humidity' },
	{ value: Service_ServiceType.PRESSURE, label: 'Pressure' },
	{ value: Service_ServiceType.POWER_METER, label: 'Power meter' },
	{ value: Service_ServiceType.CAMERA, label: 'Camera' },
	{ value: Service_ServiceType.MEDIA_PLAYBACK, label: 'Media playback' },
	{ value: Service_ServiceType.VOLUME, label: 'Volume' },
	{ value: Service_ServiceType.MEDIA_SOURCES, label: 'Media sources' },
	{ value: Service_ServiceType.REMOTE, label: 'Remote' },
	{ value: Service_ServiceType.PAIRING, label: 'Pairing' },
	{ value: Service_ServiceType.SHARING, label: 'Sharing' },
	{ value: Service_ServiceType.MODE, label: 'Mode' },
	{ value: Service_ServiceType.GENERIC, label: 'Generic' }
];

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
