<script lang="ts">
	import { cn } from "$lib/utils";
	import * as Services from '$lib/components/wh/service';
	import { Service_ServiceType } from '$lib/api/v1/clients/client_service_pb';

	let {
		class: className,
		service,
		...rest
	} : Services.StandardProps & {
		class?: string | undefined
	} = $props();
</script>

{#if
	service.typ !== Service_ServiceType.INFO &&
	service.typ !== Service_ServiceType.ONLINE
}
	<div class={cn(className)}>
		{#if service.typ == Service_ServiceType.BATTERY}
			<Services.BatteryService {service} {...rest}/>
		{:else if service.typ == Service_ServiceType.BUTTON}
			<Services.ButtonService {service} {...rest}/>
		{:else if service.typ == Service_ServiceType.CLIMATE}
			<Services.ClimateService {service} {...rest}/>
		{:else if service.typ == Service_ServiceType.ENUM}
			<Services.EnumService {service} {...rest}/>
		{:else if service.typ == Service_ServiceType.ENVIRONMENT}
			<Services.EnvironmentService {service} {...rest}/>
		{:else if service.typ == Service_ServiceType.LIGHTBULB}
			<Services.LightbulbService {service} {...rest}/>
		{:else}
			<Services.ServiceRoot
				{service}
				{...rest}
				actionPending={false}
				errorSignal={null}>
			</Services.ServiceRoot>
		{/if}
	</div>
{/if}
