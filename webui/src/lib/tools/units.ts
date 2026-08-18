import { Unit } from '$lib/api/v1/clients/client_service_pb';

// unitLabel returns a display suffix for a value with the given unit, or an
// empty string when the unit is undefined or unknown.
export function unitLabel(unit: Unit): string {
	switch (unit) {
		case Unit.PERCENTAGE:
			return '%';
		case Unit.ARC_DEGREES:
			return '°';
		case Unit.CELSIUS:
			return '°C';
		case Unit.LUX:
			return ' lux';
		case Unit.SECONDS:
			return 's';
		case Unit.PPM:
			return ' PPM';
		case Unit.MICROGRAMS_PER_CUBIC_METER:
			return ' µg/m³';
		case Unit.VOLTS:
			return 'V';
		case Unit.AMPS:
			return 'A';
		case Unit.WATTS:
			return 'W';
		case Unit.MIREDS:
			return ' mireds';
		case Unit.HECTOPASCAL:
			return ' hPa';
	}
	return '';
}
