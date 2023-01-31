<script lang="ts">
	import classNames from 'classnames';
	import { NumberValue } from '../../api/value_pb';
	import { DeviceValue } from '../../api/device_pb';

	export let value: NumberValue = null;
	export let writable: boolean = false;
	export let writer: (value: DeviceValue) => void = null;
	$: value_ = value;

	let editing = false;
	let edit = 0;

	function sendClicked() {
		if (writable) {
			const vv = new NumberValue();
			vv.setValue(edit);
			const v = new DeviceValue();
			v.setNumber(vv);
			writer(v);
		}
	}
</script>

{#if writable}
	<span class="field has-addons">
		<span class="control">
			<input
				class="input"
				type="number"
				placeholder={value_.getValue().toString()}
				value={editing ? edit : value_.getValue()}
				on:focus={() => { edit = value.getValue(); }}
				on:input={(event) => { editing = true; edit = event.target.value; }}
			>
			<!-- <input type="range" min="0" max="1" value={0.5} step="0.1"/> -->
		</span>
		<span class="control">
			<button class="button" on:click={sendClicked} disabled={!editing}>
				<span class="icon is-small">
					<!-- https://ionic.io/ionicons -->
					<svg xmlns="http://www.w3.org/2000/svg" class="ionicon" viewBox="0 0 512 512"><title>Send</title><path d="M470.3 271.15L43.16 447.31a7.83 7.83 0 01-11.16-7V327a8 8 0 016.51-7.86l247.62-47c17.36-3.29 17.36-28.15 0-31.44l-247.63-47a8 8 0 01-6.5-7.85V72.59c0-5.74 5.88-10.26 11.16-8L470.3 241.76a16 16 0 010 29.39z" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"/></svg>
				</span>
			</button>
		</span>
		<span class="control">
			<button class="button" on:click={() => { editing = false; }} disabled={!editing}>
				<span class="icon is-small">
					<!-- https://ionic.io/ionicons -->
					<svg xmlns="http://www.w3.org/2000/svg" class="ionicon" viewBox="0 0 512 512"><title>Trash</title><path d="M112 112l20 320c.95 18.49 14.4 32 32 32h184c17.67 0 30.87-13.51 32-32l20-320" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"/><path stroke="currentColor" stroke-linecap="round" stroke-miterlimit="10" stroke-width="32" d="M80 112h352"/><path d="M192 112V72h0a23.93 23.93 0 0124-24h80a23.93 23.93 0 0124 24h0v40M256 176v224M184 176l8 224M328 176l-8 224" fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="32"/></svg>
				</span>
			</button>
		</span>
	</span>
{:else}
	<span>{value.getValue()}</span>
{/if}

<style>
	.input {
		max-width: 6em;
	}
	.ionicon {
		width: 80%;
		height: 80%;
		display: block;
	}
</style>
