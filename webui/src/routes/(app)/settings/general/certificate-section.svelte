<script lang="ts">
	import { buttonVariants } from '$lib/components/ui/button';
	import * as Field from '$lib/components/ui/field/index.js';
</script>

<Field.Set>
	<Field.Legend>Certificate</Field.Legend>
	<Field.Description>
		Devices can trust this server's local certificate authority to stop browser warnings. Share the link below on
		each device.
	</Field.Description>

	<Field.Field>
		<a href="/trust" class={buttonVariants({ variant: 'outline' })}>Trust this server</a>
	</Field.Field>
</Field.Set>
