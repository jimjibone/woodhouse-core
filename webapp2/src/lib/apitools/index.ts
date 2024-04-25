import { Device, Service_ServiceType } from '$lib/api/v1/clients/client_service_pb';

export const getDeviceName = (dev: Device): string => {
    for (const ser of dev.services) {
        if (ser.typ === Service_ServiceType.INFO) {
            for (const attr of ser.attrs) {
                if (attr.id === "name") {
                    return attr.text!.value;
                }
            }
        }
    }
    return dev.id;
}

export interface DeviceInfo {
    name: string;
    online: boolean;
};

export const getDeviceInfo = (dev: Device): DeviceInfo => {
    const info = {} as DeviceInfo;
    info.name = dev.id;
    for (const ser of dev.services) {
        if (ser.typ === Service_ServiceType.INFO) {
            for (const attr of ser.attrs) {
                if (attr.id === "name") {
                    info.name = attr.text!.value;
                    break;
                }
            }
        }
        if (ser.typ === Service_ServiceType.ONLINE) {
            for (const attr of ser.attrs) {
                if (attr.id === "online") {
                    info.online = attr.bool!.value;
                    break;
                }
            }
        }
    }
    return info;
}
