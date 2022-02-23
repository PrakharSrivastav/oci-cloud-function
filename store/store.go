package store

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
	"log"
	"strings"
	"sync"
	"unicode/utf8"
)

func (c *Client) GetScheduledSteps(ctx context.Context, id int64) ([]ScheduledSteps, error) {
	q := "select * from admin.schedule_steps where sch_id = :1"
	var ss []ScheduledSteps
	err := c.conn.SelectContext(ctx, &ss, q, id)
	if err != nil {
		return nil, err
	}
	return ss, err
}

func (c *Client) AddScheduledHistory(ctx context.Context, hh []ScheduledHistory) error {
	txx, err := c.conn.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func(txx *sqlx.Tx) { _ = txx.Commit() }(txx)

	stmt, err := txx.PreparexContext(ctx, "insert into admin.SCHEDULE_HISTORY (SCH_ID, SEQ, STATUS, DESCRIPTION,TYPE ) values (:1,:2,:3,:4,:5)")
	if err != nil {
		_ = txx.Rollback()
		return err
	}
	defer func(stmt *sqlx.Stmt) { _ = stmt.Close() }(stmt)

	for i := range hh {
		if _, err := stmt.ExecContext(ctx, hh[i].SchID, hh[i].Seq, hh[i].Status, hh[i].Description, hh[i].Type); err != nil {
			_ = txx.Rollback()
			return err
		}
	}
	return nil
}

func (c *Client) GetScheduledJobByIDAndName(ctx context.Context, schID int64, name string) (*ScheduledHistory, error) {
	sh := ScheduledHistory{}
	err := c.conn.GetContext(ctx, &sh, "select * from admin.SCHEDULE_HISTORY where SCH_ID = :1 and TYPE = :2 ", schID, name)
	if err != nil {
		return nil, err
	}
	return &sh, nil
}

func (c *Client) GetScheduledJobByID(ctx context.Context, schID int64, name string) (*ScheduledHistory, error) {
	sh := ScheduledHistory{}
	err := c.conn.GetContext(ctx, &sh, "select * from admin.SCHEDULE_HISTORY where SCH_ID = :1 and TYPE = :2 ", schID, name)
	if err != nil {
		return nil, err
	}
	return &sh, nil
}

func (c *Client) UpdateScheduledJobStatus(ctx context.Context, ID int64, status, description string) error {

	txx, err := c.conn.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}

	stmt, err := txx.PrepareContext(ctx, "update admin.SCHEDULE_HISTORY set status = :1, description = :2 where id = :3")
	if err != nil {
		txx.Rollback()
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, status, sql.NullString{String: description, Valid: true}, ID)
	if err != nil {
		txx.Rollback()
		return err
	}

	return txx.Commit()
}

func (c *Client) SaveBatch(list []string, wg *sync.WaitGroup, ch chan struct{}) {
	query := `Insert into EXT_MVR_TEXT_DATA VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9,
:10, :11, :12, :13, :14, :15, :16, :17, :18, :19,
:20, :21, :22, :23, :24, :25, :26, :27, :28, :29,
:30, :31, :32, :33, :34, :35, :36, :37, :38, :39,
:40, :41, :42, :43, :44, :45, :46, :47, :48, :49,
:50, :51, :52, :53, :54, :55, :56, :57, :58, :59,
:60, :61, :62, :63, :64, :65, :66, :67, :68, :69,
:70, :71, :72, :73, :74, :75, :76, :77, :78, :79,
:80, :81, :82, :83, :84, :85 )  `

	defer wg.Done()
	ch <- struct{}{}
	tx, err := c.conn.Beginx()
	if err != nil {
		fmt.Println("cannot start tx", err)
		<-ch
		return
	}
	defer func(tx *sqlx.Tx) { _ = tx.Commit() }(tx)

	stmt, err := tx.Prepare(query)
	if err != nil {
		fmt.Println("conn.Prepare: ", err)
		tx.Rollback()
		<-ch
		return
	}
	defer func(stmt *sql.Stmt) { _ = stmt.Close() }(stmt)

	for i := range list {
		rr := []rune(list[i])
		for i := 0; i < len(rr); i++ {
			if utf8.RuneLen(rr[i]) == 3 || utf8.RuneLen(rr[i]) == 2 {
				ss := string(rr[i])
				if ss != "Ø" && ss != "ø" && ss != "Æ" && ss != "æ" && ss != "Å" && ss != "å" && ss != "Ä" && ss != "ä" && ss != "Ö" && ss != "ö" {
					rr[i] = ';'
				}
			}
		}
		split := strings.Split(strings.TrimSpace(string(rr)), ";")
		if len(split) != 85 {
			log.Println("skipping")
			continue
		}

		if _, err = stmt.Exec(split[0], split[1], split[2], split[3], split[4], split[5], split[6], split[7], split[8], split[9],
			split[10], split[11], split[12], split[13], split[14], split[15], split[16], split[17], split[18], split[19],
			split[20], split[21], split[22], split[23], split[24], split[25], split[26], split[27], split[28], split[29],
			split[30], split[31], split[32], split[33], split[34], split[35], split[36], split[37], split[38], split[39],
			split[40], split[41], split[42], split[43], split[44], split[45], split[46], split[47], split[48], split[49],
			split[50], split[51], split[52], split[53], split[54], split[55], split[56], split[57], split[58], split[59],
			split[60], split[61], split[62], split[63], split[64], split[65], split[66], split[67], split[68], split[69],
			split[70], split[71], split[72], split[73], split[74], split[75], split[76], split[77], split[78], split[79],
			split[80], split[81], split[82], split[83], split[84]); err != nil {
			fmt.Println("stmt.Exec: ", err)
			tx.Rollback()
			<-ch
			return
		}
	}
	<-ch
	return
}
