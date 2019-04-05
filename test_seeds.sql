insert into `servers` (`ip`, `port`, `description`) values
    (123, 123, '123'),
    (456, 456, '456'),
    (789, 789, '789');

insert into `users` (`name`, `pubkey`) values
    ('miu', '+ae6b173130fb836b5260ae11e30bee912e1ab0b35916c843'),
    ('Frosty', '+99be4d5a3f77076eb20978797d43bd9ccf94b39e03a5cb2'),
    ('Tagger', '+a1e915b99c0a2cc5a1b3590377e04bdc490f46fab7a7fc00'),
    ('Ignis', '+1e67875ae6107de18f006bebc34e9e44f795e35e67e4d9ca'),
    ('Murrr', '-8836a84f75db88f28b4dda4394ccd3ad77a1a556cc58169f'),
    ('Redon', '-efa043131ca8e8f68ed98cfcc069ee2d8fac00f0a5b523f7');

insert into `games` (`server`, `mode`, `map`, `ended_at`) values
    (123, 0, 'ot', 1),
    (123, 0, 'ot', 2),
    (123, 5, 'ot', 3),
    (456, 5, 'kffa', 1),
    (456, 0, 'kffa', 2),
    (456, 0, 'turbine', 3),
    (789, 0, 'turbine', 1),
    (789, 0, 'turbine', 2),
    (789, 3, 'turbine', 3),
    (789, 3, 'turbine', 4);

insert into `stats` (`game`, `user`, `frags`, `deaths`, `damage`, `potential`, `flags`) values
    (1, 'miu', 1, 1, 100, 100, 0),
    (1, 'Frosty', 1, 1, 100, 100, 0),
    (2, 'miu', 2, 2, 100, 100, 0),
    (2, 'Frosty', 2, 2, 100, 100, 0),
    (4, 'Tagger', 7, 8, 100, 900, 0),
    (4, 'Ignis', 7, 8, 100, 900, 0),
    (4, 'Murrr', 7, 8, 100, 900, 0);