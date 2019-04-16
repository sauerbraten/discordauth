create table `clans` (
	`name` text primary key,
	`created_at` integer not null default (strftime('%s', 'now'))
);

alter table `users` add column `clan` text default null references `clans`(`name`);