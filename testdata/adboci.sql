CREATE TABLE schedule (
    id         NUMBER
        GENERATED ALWAYS AS IDENTITY,
    name       VARCHAR2(200) NOT NULL,
    type       VARCHAR2(50) NOT NULL,
    created_on TIMESTAMP DEFAULT systimestamp,
    CONSTRAINT schedule_pk PRIMARY KEY ( id ) ENABLE
);

-- insert into admin.schedule (name,type) values ('mvr-tech-data-import','FTP2DB');
-- select * from admin.schedule;

CREATE TABLE schedule_steps (
    id          NUMBER
        GENERATED ALWAYS AS IDENTITY,
    sch_id      NUMBER NOT NULL,
    seq         NUMBER NOT NULL,
    type        VARCHAR2(50),
    description VARCHAR2(100),
    CONSTRAINT schedule_steps_pk PRIMARY KEY ( id ) ENABLE,
    CONSTRAINT fk_sch_id FOREIGN KEY ( sch_id )
        REFERENCES schedule ( id )
);
/*
insert into admin.schedule_steps (sch_id, seq, type) VALUES (1,1,'Overall');
insert into admin.schedule_steps (sch_id, seq, type) VALUES (1,10,'Schedule');
insert into admin.schedule_steps (sch_id, seq, type) VALUES (1,20,'DownloadFromBucket');
insert into admin.schedule_steps (sch_id, seq, type) VALUES (1,30,'UnzipData');
insert into admin.schedule_steps (sch_id, seq, type) VALUES (1,40,'WriteToDatabase');
insert into admin.schedule_steps (sch_id, seq, type) VALUES (1,50,'CleanupData');
*/
SELECT
    *
FROM
    admin.schedule_steps;

CREATE TABLE schedule_history (
    id          NUMBER GENERATED ALWAYS AS IDENTITY,
    sch_id      NUMBER NOT NULL,
    seq         NUMBER NOT NULL,
    status      VARCHAR2(50),  -- Started / InProgress / Failed / Complete
    type        VARCHAR2(50),
    created_at  TIMESTAMP DEFAULT systimestamp,
    updated_at  TIMESTAMP,
    description VARCHAR2(255),
    CONSTRAINT schedule_history_pk PRIMARY KEY ( id ) ENABLE,
    CONSTRAINT fk_sch_hist_id FOREIGN KEY ( sch_id ) REFERENCES schedule ( id )
);


CREATE TABLE schedule_info (
    id          NUMBER GENERATED ALWAYS AS IDENTITY,

    CONSTRAINT schedule_info_pk PRIMARY KEY ( id ) ENABLE,

);

select count(1) from EXT_MVR_TEXT_DATA;


