<script lang="ts">
import { Icon } from '@lucide/svelte'
let props = $props();
const iconNode = [
    ["path", { "d": "M17.5 4.5V2H19v3.5" }],
    ["path", { "d": "m2 7 10-5 10 5" }],
    // ["path", { "d": "M4.5 10.5 6 21l6-5.5 6 5.5 1.5-10.5" }]
    ["path", { "d": "M4.5 10.5 6 22l6-6.5 6 6.5 1.5-11.5" }]
];
</script>

<Icon name="woodhouse" {...props} iconNode={iconNode}>
  {@render props.children?.()}
</Icon>
