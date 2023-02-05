CREATE TABLE IF NOT EXISTS taskresult
(
    taskid uuid    NOT NULL,
    result boolean NOT NULL,
    CONSTRAINT taskresult_pkey PRIMARY KEY (taskid)
);

CREATE TABLE IF NOT EXISTS approvetime
(
    id            serial       PRIMARY KEY,
    taskid        uuid         NOT NULL,
    approver      varchar(255),
    eventtype     varchar(10),
    starttime     timestamp,
    endtime       timestamp
);