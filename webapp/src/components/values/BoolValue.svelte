<script lang="ts">
	import classNames from 'classnames';
	import { BoolValue } from '../../api/value_pb';
	import { DeviceValue } from '../../api/device_pb';
	import { Switch } from "../../lib/components/ui/switch";

	export let value: BoolValue = null;
	export let writable: boolean = false;
	export let writer: (value: DeviceValue) => void = null;

	$: onClasses = classNames({
		"button": true,
		"is-active": value.getValue(),
	});
	$: offClasses = classNames({
		"button": true,
		"is-active": !value.getValue(),
	});

	function onToggle() {
		if (writable) {
			const vv = new BoolValue();
			vv.setValue(!value.getValue());
			const v = new DeviceValue();
			v.setBool(vv);
			writer(v);
		}
	}
	function onClicked() {
		if (writable) {
			const vv = new BoolValue();
			vv.setValue(true);
			const v = new DeviceValue();
			v.setBool(vv);
			writer(v);
		}
	}
	function offClicked() {
		if (writable) {
			const vv = new BoolValue();
			vv.setValue(false);
			const v = new DeviceValue();
			v.setBool(vv);
			writer(v);
		}
	}
</script>

{#if writable}
	<span class="field has-addons">
		<!-- <span class="control">
			<button class={onClasses} on:click={onClicked}>
				On
			</button>
		</span>
		<span class="control">
			<button class={offClasses} on:click={offClicked}>
				Off
			</button>
		</span> -->
		<Switch checked={value.getValue()} onCheckedChange={onToggle} />
	</span>
{:else}
	<span>{value.getValue() ? "True" : "False"}</span>
{/if}
