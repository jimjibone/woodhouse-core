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
		{#if service.typ == Service_ServiceType.BATTERY}
			<Services.BatteryService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.BUTTON}
			<Services.ButtonService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.CAMERA}
			<Services.CameraService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.CONTACT}
			<Services.ContactService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.COVER}
			<Services.CoverService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.MOTION}
			<Services.MotionService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.PRESENCE}
			<Services.PresenceService {service} {...rest} />
		{:else if service.typ == Service_ServiceType.UPDATE}
			<Services.UpdateService {service} {...rest} />
		{:else}
			<Services.GenericService {service} {...rest} />
		{/if}
	</div>
{/if}
