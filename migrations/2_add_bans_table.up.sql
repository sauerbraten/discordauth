create table `bans` (
	`name` text primary key,
	`created_at` integer not null default (strftime('%s', 'now'))
);
