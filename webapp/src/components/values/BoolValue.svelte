<script lang="ts">
	import classNames from 'classnames';
	import { BoolValue } from '../../api/value_pb';
	import { DeviceValue } from '../../api/device_pb';
	import { Switch } from "../../lib/components/ui/switch";

	export let value: BoolValue = null;
	export let disabled: boolean = false;
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
		if (!disabled && writer) {
			const vv = new BoolValue();
			vv.setValue(!value.getValue());
			const v = new DeviceValue();
			v.setBool(vv);
			writer(v);
		}
	}
	function onClicked() {
		if (!disabled && writer) {
			const vv = new BoolValue();
			vv.setValue(true);
			const v = new DeviceValue();
			v.setBool(vv);
			writer(v);
		}
	}
	function offClicked() {
		if (!disabled && writer) {
			const vv = new BoolValue();
			vv.setValue(false);
			const v = new DeviceValue();
			v.setBool(vv);
			writer(v);
		}
	}
</script>

<span class="field has-addons">
	<Switch checked={value.getValue()} onCheckedChange={onToggle} disabled={disabled} />
</span>
