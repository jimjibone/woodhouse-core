// Framework-free helpers for the v2 device model:
// Device -> Endpoint -> Service -> Attribute, plus typed readers and builders
// for the Value oneof.

import { create } from '@bufbuild/protobuf';
import {
	ColorValueSchema,
	FieldValueSchema,
	ListValueSchema,
	StructValueSchema,
	TimeValueSchema,
	ValueSchema,
	Permissions,
	Unit,
	type Attribute,
	type ColorValue,
	type CommandSpec,
	type Endpoint,
	type FieldSpec,
	type FieldValue,
	type ListValue,
	type Service,
	type Spec,
	type StructValue,
	type TimeValue,
	type Value
} from '$lib/api/v2/clients/client_service_pb';
import { unitLabel } from '$lib/tools/units';

export const ROOT_ENDPOINT_ID = '0';

//****************************** Endpoints and services ******************************//

type HasEndpoints = { endpoints: Endpoint[] };

export const isRootEndpoint = (ep: Endpoint | string): boolean =>
	(typeof ep === 'string' ? ep : ep.id) === ROOT_ENDPOINT_ID;

// Endpoint IDs are strings but are often numeric, so compare numerically when
// both are numbers ("2" before "10") and lexically otherwise.
const compareEndpointIDs = (a: string, b: string): number => {
	const an = /^\d+$/.test(a);
	const bn = /^\d+$/.test(b);
	if (an && bn) return Number(a) - Number(b);
	return a < b ? -1 : a > b ? 1 : 0;
};

/** Endpoints with the root endpoint first, then the others ordered by ID. */
export const sortedEndpoints = (endpoints: readonly Endpoint[]): Endpoint[] =>
	[...endpoints].sort((a, b) => {
		if (isRootEndpoint(a) !== isRootEndpoint(b)) return isRootEndpoint(a) ? -1 : 1;
		return compareEndpointIDs(a.id, b.id);
	});

export const rootEndpoint = (dev: HasEndpoints): Endpoint | undefined =>
	dev.endpoints.find((ep) => ep.id === ROOT_ENDPOINT_ID);

export const findEndpoint = (dev: HasEndpoints, endpointID: string): Endpoint | undefined =>
	dev.endpoints.find((ep) => ep.id === endpointID);

export const findService = (dev: HasEndpoints, endpointID: string, serviceID: string): Service | undefined =>
	findEndpoint(dev, endpointID)?.services.find((srv) => srv.id === serviceID);

/** Every (endpoint, service) pair of a device, root endpoint first. */
export function* iterServices(dev: HasEndpoints): Generator<{ endpoint: Endpoint; service: Service }> {
	for (const endpoint of sortedEndpoints(dev.endpoints)) {
		for (const service of endpoint.services) {
			yield { endpoint, service };
		}
	}
}

/**
 * The label to show for an endpoint: the user's label, else the bridge's.
 * When neither is set the root endpoint has no label (it is the device
 * itself, so the result is ''), and other endpoints fall back to
 * "Endpoint <id>".
 */
export const endpointLabel = (ep: Pick<Endpoint, 'id' | 'label' | 'userLabel'>): string => {
	if (ep.userLabel) return ep.userLabel;
	if (ep.label) return ep.label;
	return ep.id === ROOT_ENDPOINT_ID ? '' : `Endpoint ${ep.id}`;
};

//****************************** Attributes and commands ******************************//

export const findAttr = (service: Service, id: string): Attribute | undefined =>
	service.attrs.find((attr) => attr.id === id);

export const attrValue = (service: Service, id: string): Value | undefined => findAttr(service, id)?.value;

export const findCommand = (service: Service, id: string): CommandSpec | undefined =>
	service.commands.find((cmd) => cmd.id === id);

export const isWritable = (attr: Attribute): boolean => attr.perms === Permissions.PERM_READWRITE;

//****************************** Value readers ******************************//

export const valueKind = (v: Value | undefined) => v?.kind.case;

export const asBool = (v: Value | undefined): boolean | undefined => (v?.kind.case === 'bool' ? v.kind.value : undefined);

export const asInt = (v: Value | undefined): bigint | undefined => (v?.kind.case === 'int' ? v.kind.value : undefined);

export const asFloat = (v: Value | undefined): number | undefined =>
	v?.kind.case === 'float' ? v.kind.value : undefined;

/** An int, float or duration (millis) as a number. */
export const asNumber = (v: Value | undefined): number | undefined => {
	switch (v?.kind.case) {
		case 'int':
		case 'duration':
			return Number(v.kind.value);
		case 'float':
			return v.kind.value;
	}
	return undefined;
};

export const asText = (v: Value | undefined): string | undefined => (v?.kind.case === 'text' ? v.kind.value : undefined);

/** Duration in milliseconds. */
export const asDuration = (v: Value | undefined): bigint | undefined =>
	v?.kind.case === 'duration' ? v.kind.value : undefined;

export const asTimeValue = (v: Value | undefined): TimeValue | undefined =>
	v?.kind.case === 'time' ? v.kind.value : undefined;

export const timeToDate = (t: TimeValue): Date => new Date(Number(t.seconds) * 1000 + Math.floor(t.nanos / 1_000_000));

/** The time as a Date, or undefined when it is unset or the zero time. */
export const asTime = (v: Value | undefined): Date | undefined => {
	const t = asTimeValue(v);
	if (t === undefined || (t.seconds === 0n && t.nanos === 0)) return undefined;
	return timeToDate(t);
};

export const asEnum = (v: Value | undefined): string | undefined => (v?.kind.case === 'enum' ? v.kind.value : undefined);

export const asColor = (v: Value | undefined): ColorValue | undefined =>
	v?.kind.case === 'color' ? v.kind.value : undefined;

export const asList = (v: Value | undefined): ListValue | undefined => (v?.kind.case === 'list' ? v.kind.value : undefined);

export const asStruct = (v: Value | undefined): StructValue | undefined =>
	v?.kind.case === 'struct' ? v.kind.value : undefined;

/** The value of a field of a struct value (or of a list of FieldValues). */
export const fieldValue = (fields: readonly FieldValue[] | undefined, id: string): Value | undefined =>
	fields?.find((f) => f.id === id)?.value;

// Shortcuts: read an attribute of a service as a given kind.
export const attrBool = (service: Service, id: string) => asBool(attrValue(service, id));
export const attrInt = (service: Service, id: string) => asInt(attrValue(service, id));
export const attrFloat = (service: Service, id: string) => asFloat(attrValue(service, id));
export const attrNumber = (service: Service, id: string) => asNumber(attrValue(service, id));
export const attrText = (service: Service, id: string) => asText(attrValue(service, id));
export const attrDuration = (service: Service, id: string) => asDuration(attrValue(service, id));
export const attrTime = (service: Service, id: string) => asTime(attrValue(service, id));
export const attrEnum = (service: Service, id: string) => asEnum(attrValue(service, id));

//****************************** Spec helpers ******************************//

export const specKind = (spec: Spec | undefined) => spec?.kind.case;

export const specUnit = (spec: Spec | undefined): Unit => {
	switch (spec?.kind.case) {
		case 'int':
		case 'float':
			return spec.kind.value.unit;
	}
	return Unit.UNDEFINED;
};

export type SpecRange = { min?: number; max?: number; step?: number };

/** Min, max and step of an int, float or duration spec as numbers. */
export const specRange = (spec: Spec | undefined): SpecRange => {
	switch (spec?.kind.case) {
		case 'int':
		case 'float':
		case 'duration': {
			const s = spec.kind.value;
			return {
				min: s.min === undefined ? undefined : Number(s.min),
				max: s.max === undefined ? undefined : Number(s.max),
				step: s.step === undefined ? undefined : Number(s.step)
			};
		}
	}
	return {};
};

export const specEnumOptions = (spec: Spec | undefined): string[] =>
	spec?.kind.case === 'enum' ? [...spec.kind.value.options] : [];

export const specListItem = (spec: Spec | undefined): Spec | undefined =>
	spec?.kind.case === 'list' ? spec.kind.value.item : undefined;

export const specStructFields = (spec: Spec | undefined): FieldSpec[] =>
	spec?.kind.case === 'struct' ? spec.kind.value.fields : [];

//****************************** Value builders ******************************//

export const boolValue = (v: boolean): Value => create(ValueSchema, { kind: { case: 'bool', value: v } });

export const intValue = (v: bigint | number): Value =>
	create(ValueSchema, { kind: { case: 'int', value: BigInt(Math.trunc(Number(v))) } });

export const floatValue = (v: number): Value => create(ValueSchema, { kind: { case: 'float', value: v } });

export const textValue = (v: string): Value => create(ValueSchema, { kind: { case: 'text', value: v } });

/** A duration in milliseconds. */
export const durationValue = (ms: bigint | number): Value =>
	create(ValueSchema, { kind: { case: 'duration', value: BigInt(Math.round(Number(ms))) } });

/** A time from a Date or epoch milliseconds. */
export const timeValue = (t: Date | number): Value => {
	const ms = t instanceof Date ? t.getTime() : t;
	const seconds = Math.floor(ms / 1000);
	const nanos = Math.round((ms - seconds * 1000) * 1_000_000);
	return create(ValueSchema, {
		kind: { case: 'time', value: create(TimeValueSchema, { seconds: BigInt(seconds), nanos }) }
	});
};

export const colorHueSatValue = (hue: number, sat: number): Value =>
	create(ValueSchema, {
		kind: { case: 'color', value: create(ColorValueSchema, { hueSat: { hue, sat } }) }
	});

export const colorXYValue = (x: number, y: number): Value =>
	create(ValueSchema, {
		kind: { case: 'color', value: create(ColorValueSchema, { xy: { x, y } }) }
	});

export const enumValue = (v: string): Value => create(ValueSchema, { kind: { case: 'enum', value: v } });

export const listValue = (items: Value[]): Value =>
	create(ValueSchema, { kind: { case: 'list', value: create(ListValueSchema, { items }) } });

export const structValue = (fields: FieldValue[]): Value =>
	create(ValueSchema, { kind: { case: 'struct', value: create(StructValueSchema, { fields }) } });

/** Pairs an ID with a value for a write or a command parameter. */
export const field = (id: string, value: Value): FieldValue => create(FieldValueSchema, { id, value });

//****************************** Formatting ******************************//

const formatDurationMs = (ms: number): string => {
	if (Math.abs(ms) < 1000) return `${ms}ms`;
	const secs = ms / 1000;
	if (Math.abs(secs) < 60) return `${+secs.toFixed(2)}s`;
	const total = Math.round(secs);
	const h = Math.floor(total / 3600);
	const m = Math.floor((total % 3600) / 60);
	const s = total % 60;
	return [h > 0 ? `${h}h` : '', m > 0 ? `${m}m` : '', s > 0 || (h === 0 && m === 0) ? `${s}s` : '']
		.filter((p) => p !== '')
		.join(' ');
};

/**
 * A short single-line rendering of a value. Lists and structs are summarised
 * inline; use a table renderer for anything bigger. The spec, when given,
 * supplies the unit.
 */
export const formatValue = (v: Value | undefined, spec?: Spec): string => {
	switch (v?.kind.case) {
		case undefined:
			return '';
		case 'bool':
			return v.kind.value ? 'Yes' : 'No';
		case 'int':
			return v.kind.value.toLocaleString() + unitLabel(specUnit(spec));
		case 'float':
			return v.kind.value.toLocaleString(undefined, { maximumFractionDigits: 2 }) + unitLabel(specUnit(spec));
		case 'text':
			return v.kind.value;
		case 'duration':
			return formatDurationMs(Number(v.kind.value));
		case 'time': {
			const t = asTime(v);
			return t ? t.toLocaleString() : 'Not set';
		}
		case 'color': {
			const c = v.kind.value;
			if (c.hueSat) return `Hue ${c.hueSat.hue.toFixed(0)}, Sat ${c.hueSat.sat.toFixed(0)}%`;
			if (c.xy) return `x ${c.xy.x.toFixed(3)}, y ${c.xy.y.toFixed(3)}`;
			return '';
		}
		case 'enum':
			return v.kind.value;
		case 'image':
			return '(image)';
		case 'list': {
			const items = v.kind.value.items;
			const itemSpec = specListItem(spec);
			return `[${items.map((i) => formatValue(i, itemSpec)).join(', ')}]`;
		}
		case 'struct': {
			const fields = specStructFields(spec);
			return `{${v.kind.value.fields
				.map((f) => `${f.id}: ${formatValue(f.value, fields.find((fs) => fs.id === f.id)?.spec)}`)
				.join(', ')}}`;
		}
	}
	return '';
};
