create table `users` (
	`name` text primary key,
	`pubkey` text not null,
	`admin` integer not null default 0,
	`created_at` integer not null default (strftime('%s', 'now')),
	`last_authed_at` integer not null default 0
);
