<script lang="ts">
	import { NumberValue } from '../../api/value_pb';
	import { DeviceValue } from '../../api/device_pb';
	import { Input } from "$lib/components/ui/input";
	import { Button } from "$lib/components/ui/button";
	import { Send, Trash2 } from 'lucide-svelte';

	export let value: DeviceValue;
	export let disabled: boolean = false;
	export let writer: ((value: DeviceValue) => void) | undefined;
	$: value_ = value;

	let editing = false;
	let edit = 0;

	function sendClicked() {
		if (!disabled && writer) {
			const vv = new NumberValue();
			vv.setValue(edit);
			const v = new DeviceValue();
			v.setNumber(vv);
			writer(v);
		}
	}
</script>

{#if !disabled}
	<form class="flex w-full max-w-sm items-center space-x-2">
		<Input
			type="number"
			placeholder={value_.getValue().toString()}
			value={editing ? edit : value_.getValue()}
			on:focus={() => { edit = value.getValue(); }}
			on:input={(event) => { editing = true; edit = event.target.value; }}
		/>
		<Button on:click={sendClicked} disabled={!editing}><Send size={18}/></Button>
		<Button on:click={() => { editing = false; }} disabled={!editing}><Trash2 size={18}/></Button>
	</form>
{:else}
	<form class="flex w-full max-w-sm items-center space-x-2">
		<Input
			type="number"
			value={editing ? edit : value.getValue()}
			disabled
		/>
	</form>
{/if}
