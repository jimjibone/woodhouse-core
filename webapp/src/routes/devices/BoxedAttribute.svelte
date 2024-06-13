<script lang="ts">
	import { Attribute as AttributeType, Value } from '$lib/api/v1/clients/client_service_pb';

	import * as Card from '$lib/components/ui/card';
	import Attribute from './Attribute.svelte';

	export let online: boolean;
	export let attr: AttributeType;
	export let onAction: (vals: Value[]) => Promise<void> | undefined;
</script>

<Card.Root class={online ? '' : 'bg-muted'}>
	<Card.Header class="pb-3">
		<Card.Title>{attr.id}</Card.Title>
	</Card.Header>
	<Card.Content>
		<Attribute {online} {attr} {onAction} />
	</Card.Content>
</Card.Root>
