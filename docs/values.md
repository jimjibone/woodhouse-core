# Values

## Adding or Changing a Value Type

1. Add or change the value type in `api/value.proto`
2. Open `api/device.proto` and add the new value to the `DeviceValue` message
3. Generate protos with `task proto-all`
4. Open `apitools/value.go` and update the `ValueAs` and `ValueFrom` functions
5. Open `webapp/src/components/values/*.svelte` and update the value rendering code
6. Open `webapp/src/components/DeviceValue.svelte` and update the DeviceValue rendering code
