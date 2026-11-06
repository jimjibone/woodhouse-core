// The old v1 components (relay, lightbulb, environment, climate, input, enum)
// are still in this directory but are no longer exported or routed: their
// service types now render with GenericService until they are ported.
import BatteryService from './battery.svelte';
import ButtonService from './button.svelte';
import CameraService from './camera.svelte';
import ContactService from './contact.svelte';
import CoverService from './cover.svelte';
import GenericService from './generic.svelte';
import MotionService from './motion.svelte';
import PresenceService from './presence.svelte';
import UpdateService from './update.svelte';
import ServiceAction from './service-action.svelte';
import ServiceEnumerator from './service-enumerator.svelte';
import ServiceRoot, { type StandardProps } from './service-root.svelte';

export {
	BatteryService,
	ButtonService,
	CameraService,
	ContactService,
	CoverService,
	GenericService,
	MotionService,
	PresenceService,
	UpdateService,
	ServiceAction,
	ServiceEnumerator,
	ServiceRoot
};

export type { StandardProps };
