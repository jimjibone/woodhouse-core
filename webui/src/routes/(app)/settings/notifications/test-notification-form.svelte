<script lang="ts">
	import Button from '$lib/components/ui/button/button.svelte';
	import * as Field from '$lib/components/ui/field/index.js';
	import Input from '$lib/components/ui/input/input.svelte';
	import * as RadioGroup from '$lib/components/ui/radio-group/index.js';
	import Label from '$lib/components/ui/label/label.svelte';
	import { SendTestNotification } from '@/stores/requests';
	import {
		NotificationLevel,
		NotificationAudience_Kind,
		UserRole
	} from '$lib/api/v1/clients/user_service_pb';
	import { ConnectError } from '@connectrpc/connect';
	import { toSentenceCase } from '@/tools/headline-case';
	import { toast } from 'svelte-sonner';

	const id = $props.id();

	let updateError: string | null = $state(null);
	let submitting = $state(false);

	let title = $state('');
	let body = $state('');
	let level = $state('info');
	// Defaults to the caller alone, matching the server: pressing send on a
	// shared instance should not notify the whole household by accident.
	let audience = $state('self');
	let username = $state('');

	const levels: Record<string, NotificationLevel> = {
		info: NotificationLevel.INFO,
		success: NotificationLevel.SUCCESS,
		warning: NotificationLevel.WARNING,
		error: NotificationLevel.ERROR
	};

	const audiences = (): { kind: NotificationAudience_Kind; role?: UserRole; username?: string } => {
		switch (audience) {
			case 'everyone':
				return { kind: NotificationAudience_Kind.ALL };
			case 'admins':
				return { kind: NotificationAudience_Kind.ROLE, role: UserRole.ADMIN };
			case 'users':
				return { kind: NotificationAudience_Kind.ROLE, role: UserRole.USER };
			case 'named':
				return { kind: NotificationAudience_Kind.USER, username: username.trim() };
			default:
				// The server fills in the calling admin's own username.
				return { kind: NotificationAudience_Kind.UNSPECIFIED };
		}
	};

	async function handleSubmit(event: SubmitEvent) {
		event.preventDefault();
		updateError = null;

		if (audience === 'named' && username.trim() === '') {
			updateError = 'username must not be empty';
			return;
		}

		submitting = true;
		try {
			const res = await SendTestNotification({
				title: title.trim() || undefined,
				body: body.trim() || undefined,
				level: levels[level],
				audience: audiences()
			});
			if (res instanceof ConnectError) {
				updateError = res.rawMessage;
				return;
			}
			// Deliberately not a preview of the notification itself - if delivery
			// works, the real one arrives on the stream a moment later, and that
			// is the thing being tested.
			toast.success('Test notification sent');
		} finally {
			submitting = false;
		}
	}
</script>

<form onsubmit={handleSubmit}>
	<Field.Set>
		<Field.Legend>Send a test notification</Field.Legend>
		<Field.Description>
			Raises a notification so you can check it arrives. It is delivered exactly like a real one -
			to the bell here, and to the Woodhouse app on any signed-in device. Nothing else raises
			notifications yet.
		</Field.Description>

		<Field.Group>
			<Field.Field>
				<Field.Label for="title-{id}">Title</Field.Label>
				<Input
					id="title-{id}"
					type="text"
					autocomplete="off"
					maxlength={200}
					placeholder="Test notification"
					bind:value={title}
				/>
			</Field.Field>

			<Field.Field>
				<Field.Label for="body-{id}">Body</Field.Label>
				<Input
					id="body-{id}"
					type="text"
					autocomplete="off"
					maxlength={2000}
					placeholder="If you can see this, notification delivery is working."
					bind:value={body}
				/>
			</Field.Field>

			<Field.Field>
				<Field.Label>Level</Field.Label>
				<RadioGroup.Root bind:value={level} class="gap-2">
					{#each [['info', 'Info'], ['success', 'Success'], ['warning', 'Warning'], ['error', 'Error']] as [value, label] (value)}
						<div class="flex items-center gap-2">
							<RadioGroup.Item value={value} id="level-{value}-{id}" class="cursor-pointer" />
							<Label for="level-{value}-{id}" class="cursor-pointer font-normal">{label}</Label>
						</div>
					{/each}
				</RadioGroup.Root>
			</Field.Field>

			<Field.Field>
				<Field.Label>Send to</Field.Label>
				<RadioGroup.Root bind:value={audience} class="gap-2">
					{#each [['self', 'Just me'], ['everyone', 'Everyone'], ['admins', 'Admins'], ['users', 'Users'], ['named', 'A specific user']] as [value, label] (value)}
						<div class="flex items-center gap-2">
							<RadioGroup.Item value={value} id="audience-{value}-{id}" class="cursor-pointer" />
							<Label for="audience-{value}-{id}" class="cursor-pointer font-normal">{label}</Label>
						</div>
					{/each}
				</RadioGroup.Root>
				{#if audience === 'named'}
					<Input
						id="username-{id}"
						type="text"
						autocomplete="off"
						placeholder="username"
						bind:value={username}
					/>
				{/if}
			</Field.Field>
		</Field.Group>

		{#if updateError}
			<Field.Error>{toSentenceCase(updateError)}</Field.Error>
		{/if}

		<Field.Field>
			<Button type="submit" class="cursor-pointer" disabled={submitting}>
				{submitting ? 'Sending…' : 'Send test notification'}
			</Button>
		</Field.Field>
	</Field.Set>
</form>
