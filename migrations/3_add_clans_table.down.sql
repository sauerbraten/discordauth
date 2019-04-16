pragma foreign_keys=off;

begin transaction;

create table `_users` (
	`name` text primary key,
	`pubkey` text not null,
	`admin` integer not null default 0,
	`created_at` integer not null default (strftime('%s', 'now')),
	`last_authed_at` integer not null default 0
);

insert into `_users` select `name`, `pubkey`, `admin`, `created_at`, `last_authed_at` from `users`;

drop table `users`;

alter table `_users` rename to `users`;

drop table `clans`;

pragma foreign_key_check;

commit;

pragma foreign_keys=on;