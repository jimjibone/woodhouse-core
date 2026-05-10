<script lang="ts">
	import { cn } from '$lib/utils';
	import * as Services from '$lib/components/wh/service';
	import { Service_ServiceType } from '$lib/api/v1/clients/client_service_pb';

	let {
		class: className,
		service,
		naturalWidth = false,
		...rest
	}: Services.StandardProps & {
		class?: string | undefined;
		naturalWidth?: boolean;
	} = $props();
</script>

{#if service.typ !== Service_ServiceType.INFO && service.typ !== Service_ServiceType.ONLINE}
	<div class={cn(className, naturalWidth && 'flex-none')}>
		{#if service.typ == Service_ServiceType.BATTERY}
			<Services.BatteryService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.BUTTON}
			<Services.ButtonService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.CAMERA}
			<Services.CameraService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.CLIMATE}
			<Services.ClimateService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.CONTACT}
			<Services.ContactService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.COVER}
			<Services.CoverService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.ENUM}
			<Services.EnumService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.ENVIRONMENT}
			<Services.EnvironmentService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.INPUT}
			<Services.InputService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.LIGHTBULB}
			<Services.LightbulbService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.PRESENCE}
			<Services.PresenceService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.RELAY}
			<Services.RelayService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.UPDATE}
			<Services.UpdateService {service} {...rest} />
		{:else}
			<Services.ServiceRoot {service} {...rest} actionPending={false} errorSignal={null}></Services.ServiceRoot>
		{/if}
	</div>
{/if}
