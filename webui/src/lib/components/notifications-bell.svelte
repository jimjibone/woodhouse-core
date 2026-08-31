<script lang="ts">
	import { onDestroy } from 'svelte';
	import { goto } from '$app/navigation';
	import * as Popover from '$lib/components/ui/popover/index.js';
	import Button from '$lib/components/ui/button/button.svelte';
	import Badge from '$lib/components/ui/badge/badge.svelte';
	import Separator from '$lib/components/ui/separator/separator.svelte';
	import { BellIcon, CheckCheckIcon, XIcon } from '@lucide/svelte';
	import { NotificationLevel } from '$lib/api/v1/clients/user_service_pb';
	import type { Notification } from '$lib/api/v1/clients/user_service_pb';
	import { NotificationsStore, notificationDate } from '@/stores/notifications-stream';
	import { DismissNotification, MarkAllNotificationsRead, MarkNotificationRead } from '@/stores/requests';
	import { clock } from '@/stores/clock';
	import { toRelativeTime } from '@/tools/time';

	// Subscribed here rather than in the popover body so the badge stays live
	// while the popover is shut, and so the stream's lifecycle is tied to a
	// component that is mounted for as long as the user is logged in.
	let notifications = $state<Notification[]>([]);
	let unread = $state(0);
	let open = $state(false);

	onDestroy(
		NotificationsStore.subscribe((update) => {
			notifications = update.notifications;
			unread = update.unread;
		})
	);

	// A coloured dot rather than a coloured row: the level is a hint, and the
	// list has to stay readable when every row is a warning.
	const levelDot = (level: NotificationLevel): string => {
		switch (level) {
			case NotificationLevel.SUCCESS:
				return 'bg-green-500';
			case NotificationLevel.WARNING:
				return 'bg-amber-500';
			case NotificationLevel.ERROR:
				return 'bg-destructive';
			default:
				return 'bg-sky-500';
		}
	};

	// No optimistic update anywhere here: every one of these round-trips through
	// the stream, which is what keeps two browsers and the phone in step.
	const handleActivate = async (notification: Notification) => {
		await MarkNotificationRead(notification.id);
		if (notification.link) {
			open = false;
			goto(notification.link);
		}
	};

	const handleDismiss = async (event: MouseEvent, notification: Notification) => {
		// The dismiss button sits inside the row button, so without this the
		// row's activate handler fires too.
		event.stopPropagation();
		await DismissNotification(notification.id);
	};
</script>

<Popover.Root bind:open>
	<Popover.Trigger>
		{#snippet child({ props })}
			<Button
				{...props}
				variant="ghost"
				size="icon"
				class="relative cursor-pointer"
				aria-label={unread > 0 ? `Notifications (${unread} unread)` : 'Notifications'}
			>
				<BellIcon />
				{#if unread > 0}
					<Badge class="absolute -right-0.5 -top-0.5 h-4 min-w-4 justify-center px-1 py-0 text-[10px] leading-none">
						{unread > 9 ? '9+' : unread}
					</Badge>
				{/if}
			</Button>
		{/snippet}
	</Popover.Trigger>

	<Popover.Content class="w-[min(24rem,calc(100vw-2rem))] p-0" align="end">
		<div class="flex items-center justify-between px-4 py-3">
			<span class="text-sm font-medium">Notifications</span>
			{#if unread > 0}
				<Button
					variant="link"
					size="sm"
					class="h-auto cursor-pointer p-0 text-xs"
					onclick={() => MarkAllNotificationsRead()}
				>
					<CheckCheckIcon class="size-3" />
					Mark all read
				</Button>
			{/if}
		</div>
		<Separator />

		{#if notifications.length === 0}
			<p class="text-muted-foreground p-6 text-center text-sm">No notifications</p>
		{:else}
			<div class="max-h-96 overflow-y-auto">
				{#each notifications as notification (notification.id)}
					<div
						class="hover:bg-accent/50 group relative flex w-full items-start gap-3 border-b px-4 py-3 text-left last:border-b-0"
						class:bg-muted={!notification.read}
					>
						<button
							class="flex flex-1 cursor-pointer items-start gap-3 text-left"
							onclick={() => handleActivate(notification)}
						>
							<span class="mt-1.5 size-2 shrink-0 rounded-full {levelDot(notification.level)}"></span>
							<span class="flex-1 space-y-0.5">
								<span class="block text-sm leading-snug" class:font-medium={!notification.read}>
									{notification.title}
								</span>
								{#if notification.body}
									<span class="text-muted-foreground block text-xs leading-snug">
										{notification.body}
									</span>
								{/if}
								<span class="text-muted-foreground block text-xs">
									{toRelativeTime(notificationDate(notification), $clock)}
								</span>
							</span>
						</button>

						<Button
							variant="ghost"
							size="icon"
							class="size-6 shrink-0 cursor-pointer opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100"
							aria-label="Dismiss notification"
							onclick={(event: MouseEvent) => handleDismiss(event, notification)}
						>
							<XIcon class="size-3" />
						</Button>
					</div>
				{/each}
			</div>
		{/if}
	</Popover.Content>
</Popover.Root>
