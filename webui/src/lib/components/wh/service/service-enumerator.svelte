<script lang="ts">
	import { cn } from '$lib/utils';
	import * as Services from '$lib/components/wh/service';
	import { Service_ServiceType } from '$lib/api/v2/clients/client_service_pb';

	let {
		class: className,
		service,
		...rest
	}: Services.StandardProps & {
		class?: string | undefined;
	} = $props();
</script>

<!-- Services with a typed component get it; every other type, including the
	 generic service, is rendered from its specs by the generic component. -->
{#if service.typ !== Service_ServiceType.INFO && service.typ !== Service_ServiceType.ONLINE}
	<div class={cn(className, rest.naturalWidth && 'flex-none')}>
		{#if service.typ == Service_ServiceType.ANALOG_INPUT}
			<Services.AnalogInputService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.BATTERY}
			<Services.BatteryService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.BINARY_INPUT}
			<Services.BinaryInputService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.BUTTON}
			<Services.ButtonService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.CAMERA}
			<Services.CameraService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.CONTACT}
			<Services.ContactService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.COVER}
			<Services.CoverService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.DEVICE_TEMPERATURE}
			<Services.DeviceTemperatureService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.HUMIDITY}
			<Services.HumidityService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.IDENTIFY}
			<Services.IdentifyService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.LIGHT}
			<Services.LightService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.MEDIA_PLAYBACK}
			<Services.MediaPlaybackService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.MEDIA_SOURCES}
			<Services.MediaSourcesService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.MODE}
			<Services.ModeService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.MOTION}
			<Services.MotionService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.ON_OFF}
			<Services.OnOffService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.PAIRING}
			<Services.PairingService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.POWER_METER}
			<Services.PowerMeterService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.PRESENCE}
			<Services.PresenceService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.PRESSURE}
			<Services.PressureService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.REMOTE}
			<Services.RemoteService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.SHARING}
			<Services.SharingService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.TEMPERATURE}
			<Services.TemperatureService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.THERMOSTAT}
			<Services.ThermostatService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.UPDATE}
			<Services.UpdateService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.VOLUME}
			<Services.VolumeService {service} {...rest} />
		{:else}
			<Services.GenericService {service} {...rest} />
		{/if}
	</div>
{/if}
