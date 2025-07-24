import type { TimeAttribute, TimeValue } from "$lib/api/v1/clients/client_service_pb";

export function attributeToDate(time: TimeAttribute): Date {
    const ms = Number(time.seconds) * 1000 + Math.floor(Number(time.nanos) / 1_000_000);
    return new Date(ms);
}

export function valueToDate(time: TimeValue): Date {
    const ms = Number(time.seconds) * 1000 + Math.floor(Number(time.nanos) / 1_000_000);
    return new Date(ms);
}

export function toHumanDate(date: Date): string {
    return date.toLocaleString();
}
