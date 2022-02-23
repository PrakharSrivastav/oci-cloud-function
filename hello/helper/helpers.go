package helper

//
//import (
//	"archive/zip"
//	"bufio"
//	"bytes"
//	"context"
//	"database/sql"
//	"fmt"
//	"github.com/PrakharSrivastav/oci-cloud-function/store"
//	"github.com/jmoiron/sqlx"
//	"github.com/openzipkin/zipkin-go"
//	"io"
//	"io/ioutil"
//	"log"
//	"os"
//	"path/filepath"
//	"strings"
//	"sync"
//	"time"
//)
//
//func UnzipUploadedFile(ctx context.Context, src string, tt *zipkin.Tracer) (string, []string, error) {
//	span, _ := tt.StartSpanFromContext(ctx, "unzipUploadedFile")
//	defer span.Finish()
//
//	dest, err := ioutil.TempDir("", "mvr-*")
//	if err != nil {
//		span.Tag(string(zipkin.TagError), err.Error())
//		log.Print("cannot create temp dir", err)
//		return "", nil, err
//	}
//
//	var filenames []string
//
//	r, err := zip.OpenReader(src)
//	if err != nil {
//		span.Tag(string(zipkin.TagError), err.Error())
//		return "", filenames, err
//	}
//	defer r.Close()
//
//	for _, f := range r.File {
//
//		// Store filename/path for returning and using later on
//		fpath := filepath.Join(dest, f.Name)
//
//		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
//		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
//			return "", filenames, fmt.Errorf("%s: illegal file path", fpath)
//		}
//
//		filenames = append(filenames, fpath)
//
//		if f.FileInfo().IsDir() {
//			// Make Folder
//			os.MkdirAll(fpath, os.ModePerm)
//			continue
//		}
//
//		// Make File
//		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
//			return "", filenames, err
//		}
//
//		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
//		if err != nil {
//			span.Tag(string(zipkin.TagError), err.Error())
//			return "", filenames, err
//		}
//
//		rc, err := f.Open()
//		if err != nil {
//			span.Tag(string(zipkin.TagError), err.Error())
//			return "", filenames, err
//		}
//
//		_, err = io.Copy(outFile, rc)
//
//		// Close the file without defer to close before next iteration of loop
//		outFile.Close()
//		rc.Close()
//
//		if err != nil {
//			return "", filenames, err
//		}
//	}
//	return dest, filenames, nil
//}
//
//func SaveObjectAsZip(ctx context.Context, content io.ReadCloser, tracer *zipkin.Tracer) (string, error) {
//
//	span, _ := tracer.StartSpanFromContext(ctx, "saveObjectAsZip")
//	defer span.Finish()
//	cBuf := new(bytes.Buffer)
//	if _, err := io.Copy(cBuf, content); err != nil {
//		log.Print("read object bytes error", err)
//		return "", err
//	}
//
//	zipFile, err := ioutil.TempFile("", "mvr-*.zip")
//	if err != nil {
//		span.Tag(string(zipkin.TagError), err.Error())
//		log.Print("can not create temp dir", err)
//		return "", err
//	}
//	defer zipFile.Close()
//
//	_, err = io.Copy(zipFile, cBuf)
//	if err != nil {
//		span.Tag(string(zipkin.TagError), err.Error())
//		return "", err
//	}
//
//	return zipFile.Name(), nil
//}
//
//func parseDataFile(ctx context.Context, path string) (int, error) {
//
//	batchSize := 200
//
//	file, err := os.Open(path)
//	if err != nil {
//		return 0, err
//	}
//	defer file.Close()
//
//	fileBuffer := bufio.NewReader(file)
//	batch := make([]string, 0, batchSize)
//	line := ""
//
//	conn, err := store.GetConnection()
//	if err != nil {
//		return 0, err
//	}
//	defer conn.Close()
//
//	count := 0
//	wg := new(sync.WaitGroup)
//	for {
//		line, err = fileBuffer.ReadString('\n')
//		if err != nil {
//			break
//		}
//		batch = append(batch, line)
//		if len(batch) != batchSize {
//			continue
//		}
//		count++
//		wg.Add(1)
//		go handle(batch, count, conn, wg)
//		batch = make([]string, 0, batchSize)
//	}
//
//	if len(batch) > 0 {
//		wg.Add(1)
//		go handle(batch, count, conn, wg)
//	}
//
//	wg.Wait()
//	if err != nil {
//		if err == io.EOF {
//			return 1, nil
//		}
//		return 0, err
//	}
//	return 1, nil
//}
//
//func handle(list []string, num int, conn *sqlx.DB, wg *sync.WaitGroup) {
//	log.Print("batch number ", num)
//	t := time.Now()
//	defer wg.Done()
//	tx, err := conn.Beginx()
//	if err != nil {
//		fmt.Printf("cannot begin tx for batch %d : err %v \n", num, err)
//		return
//	}
//	defer func(tx *sqlx.Tx) { _ = tx.Commit() }(tx)
//
//	values := make([]interface{}, 0)
//	var sb strings.Builder
//	sb.WriteString(insertAll)
//	count := 1
//	for i := range list {
//		split := strings.Split(strings.TrimSpace(list[i]), "¤")
//		if len(split) != 85 {
//			log.Println("skipping")
//			return
//		}
//		item := make([]interface{}, 0)
//		for x := range split {
//			item = append(item, count)
//			count++
//			values = append(values, split[x])
//		}
//		sb.WriteString(fmt.Sprintf(query4, item...))
//	}
//
//	sb.WriteString(selectAll)
//	stmt, err := tx.Prepare(sb.String())
//	log.Printf("batch number %d took %d for prepare \n", num, time.Now().Sub(t).Milliseconds())
//
//	if err != nil {
//		fmt.Println("conn.Prepare: ", err)
//		tx.Rollback()
//		return
//	}
//	defer func(stmt *sql.Stmt) { _ = stmt.Close() }(stmt)
//	if _, err = stmt.Exec(values...); err != nil {
//		fmt.Println("stmt.Exec: ", err)
//		tx.Rollback()
//		return
//	}
//	log.Printf("batch number %d took %d", num, time.Now().Sub(t).Milliseconds())
//
//	return
//}
//
//func handle1(list []string, num int, conn *sqlx.DB, wg *sync.WaitGroup) {
//	t := time.Now()
//
//	log.Print("batch number ", num)
//	defer wg.Done()
//	tx, err := conn.Beginx()
//	if err != nil {
//		fmt.Printf("cannot begin tx for batch %d : err %v \n", num, err)
//		return
//	}
//	defer func(tx *sqlx.Tx) { _ = tx.Commit() }(tx)
//
//	stmt, err := tx.Prepare(query)
//	if err != nil {
//		fmt.Println("conn.Prepare: ", err)
//		tx.Rollback()
//		return
//	}
//	defer func(stmt *sql.Stmt) { _ = stmt.Close() }(stmt)
//
//	for i := range list {
//		split := strings.Split(strings.TrimSpace(list[i]), "¤")
//		if len(split) != 85 {
//			log.Println("skipping")
//			return
//		}
//
//		if _, err = stmt.Exec(split[0], split[1], split[2], split[3], split[4], split[5], split[6], split[7], split[8], split[9],
//			split[10], split[11], split[12], split[13], split[14], split[15], split[16], split[17], split[18], split[19],
//			split[20], split[21], split[22], split[23], split[24], split[25], split[26], split[27], split[28], split[29],
//			split[30], split[31], split[32], split[33], split[34], split[35], split[36], split[37], split[38], split[39],
//			split[40], split[41], split[42], split[43], split[44], split[45], split[46], split[47], split[48], split[49],
//			split[50], split[51], split[52], split[53], split[54], split[55], split[56], split[57], split[58], split[59],
//			split[60], split[61], split[62], split[63], split[64], split[65], split[66], split[67], split[68], split[69],
//			split[70], split[71], split[72], split[73], split[74], split[75], split[76], split[77], split[78], split[79],
//			split[80], split[81], split[82], split[83], split[84]); err != nil {
//			fmt.Println("stmt.Exec: ", err)
//			tx.Rollback()
//			return
//		}
//	}
//
//	log.Printf("batch number %d took %d", num, time.Now().Sub(t).Milliseconds())
//	return
//}
//
//func handle2(list []string, num int) {
//	log.Print("batch2 number ", num)
//	time.Sleep(5 * time.Second)
//}
//
//var c = `Insert into ADMIN.EXT_MVR_TEXT_DATA
//( TEKN_KJM,TEKN_KJM_PERSJ,TEKN_KJM_FARGE,TEKN_UNR,TEKN_REG_F_G,TEKN_REG_F_G_N,
// TEKN_REG_EIERSKIFTE_DATO,TEKN_REG_EIER_DATO,TEKN_AVREG_DATO,TEKN_VRAKET_DATO,
// TEKN_UTFORT_DATO,TEKN_REG_STATUS,TEKN_BRUKTIMP,TEKN_MERKE,TEKN_MERKENAVN,
// TEKN_MODELL,TEKN_TKNAVN,TEKN_TUK_VERDI,TEKN_KJTGRP,TEKN_MOTORKODE,
// TEKN_MOTORYTELSE,TEKN_SLAGVOLUM,TEKN_DRIVST,TEKN_GIRKASSE,TEKN_HYBRID,
// TEKN_HYBRID_KATEGORI,TEKN_TOTVEKT,TEKN_EGENVEKT,TEKN_VOGNTOGVEKT,
// TEKN_MAKS_TAKLAST,TEKN_THV_M_BREMS,TEKN_THV_U_BREMS,TEKN_BEL_H_FESTE,
// TEKN_LENGDE,TEKN_BREDDE,TEKN_ANTALL_DORER,TEKN_FARGE,TEKN_PABYGG,
// TEKN_SITTEPLASSERFORAN,TEKN_SITTEPLASSER_TOTALT,TEKN_AKSLER,TEKN_AKSLER_DRIFT,
// TEKN_MINAVST_MS1,TEKN_MINAVST_MS2,TEKN_DEKK_1,TEKN_DEKK_2,TEKN_DEKK_3,TEKN_DEKK_F,
// TEKN_FELG_1,TEKN_FELG_2,TEKN_FELG_3,TEKN_MILI_1,TEKN_MILI_2,TEKN_MILI_3,TEKN_HAST_1,TEKN_HAST_2,TEKN_HAST_3,
// TEKN_INNP_1,TEKN_INNP_2,TEKN_INNP_3,TEKN_SPOR_1,TEKN_SPOR_2,TEKN_SPOR_3,TEKN_LAST_1,TEKN_LAST_2,TEKN_LAST_3,
// TEKN_LUFT_1,TEKN_LUFT_2,TEKN_LUFT_3,TEKN_NGN,TEKN_EU_HOVED,TEKN_EURONORM_NY,TEKN_EU_TYPE,TEKN_EU_VARIANT,TEKN_EU_VERSJON,
// TEKN_SISTE_PKK,TEKN_NESTE_PKK,TEKN_STANDSTOY,TEKN_PARTIKKELFILTER,TEKN_NOX_UTSLIPP_MGPRKH,TEKN_NOX_UTSLIPP_GPRKWH,
// TEKN_PARTIKKELUTSLIPP,TEKN_MAALEMETODE,TEKN_CO2_UTSLIPP,TEKN_DRIVSTOFF_FORBRUK )
//VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9,
//:10, :11, :12, :13, :14, :15, :16, :17, :18, :19,
//:20, :21, :22, :23, :24, :25, :26, :27, :28, :29,
//:30, :31, :32, :33, :34, :35, :36, :37, :38, :39,
//:40, :41, :42, :43, :44, :45, :46, :47, :48, :49,
//:50, :51, :52, :53, :54, :55, :56, :57, :58, :59,
//:60, :61, :62, :63, :64, :65, :66, :67, :68, :69,
//:70, :71, :72, :73, :74, :75, :76, :77, :78, :79,
//:80, :81, :82, :83, :84, :85 )  `
//
//const (
//	insertAll = `INSERT ALL`
//	selectAll = `
//select 1 from dual`
//	query = `Insert into EXT_MVR_TEXT_DATA VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9,
//:10, :11, :12, :13, :14, :15, :16, :17, :18, :19,
//:20, :21, :22, :23, :24, :25, :26, :27, :28, :29,
//:30, :31, :32, :33, :34, :35, :36, :37, :38, :39,
//:40, :41, :42, :43, :44, :45, :46, :47, :48, :49,
//:50, :51, :52, :53, :54, :55, :56, :57, :58, :59,
//:60, :61, :62, :63, :64, :65, :66, :67, :68, :69,
//:70, :71, :72, :73, :74, :75, :76, :77, :78, :79,
//:80, :81, :82, :83, :84, :85 )  `
//
//	query2 = `Insert into EXT_MVR_TEXT_DATA VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?,?, ?, ?, ?, ?, ?, ?, ?, ?, ?,?, ?, ?, ?, ?, ?, ?, ?, ?, ?,?, ?, ?, ?, ?, ?, ?, ?, ?, ?,?, ?, ?, ?, ?, ?, ?, ?, ?, ?,?, ?, ?, ?, ?, ?, ?, ?, ?, ?,?, ?, ?, ?, ?, ?, ?, ?, ?, ?,?, ?, ?, ?, ?, ?, ?, ?, ?, ?,?, ?, ?, ?, ?, ? )  `
//	query3 = `
// INTO ADMIN.EXT_MVR_TEXT_DATA  ( TEKN_KJM,TEKN_KJM_PERSJ,TEKN_KJM_FARGE,TEKN_UNR,TEKN_REG_F_G,TEKN_REG_F_G_N,TEKN_REG_EIERSKIFTE_DATO,TEKN_REG_EIER_DATO,TEKN_AVREG_DATO,TEKN_VRAKET_DATO,TEKN_UTFORT_DATO,TEKN_REG_STATUS,TEKN_BRUKTIMP,TEKN_MERKE,TEKN_MERKENAVN,TEKN_MODELL,TEKN_TKNAVN,TEKN_TUK_VERDI,TEKN_KJTGRP,TEKN_MOTORKODE,TEKN_MOTORYTELSE,TEKN_SLAGVOLUM,TEKN_DRIVST,TEKN_GIRKASSE,TEKN_HYBRID,TEKN_HYBRID_KATEGORI,TEKN_TOTVEKT,TEKN_EGENVEKT,TEKN_VOGNTOGVEKT,TEKN_MAKS_TAKLAST,TEKN_THV_M_BREMS,TEKN_THV_U_BREMS,TEKN_BEL_H_FESTE,TEKN_LENGDE,TEKN_BREDDE,TEKN_ANTALL_DORER,TEKN_FARGE,TEKN_PABYGG,TEKN_SITTEPLASSERFORAN,TEKN_SITTEPLASSER_TOTALT,TEKN_AKSLER,TEKN_AKSLER_DRIFT,TEKN_MINAVST_MS1,TEKN_MINAVST_MS2,TEKN_DEKK_1,TEKN_DEKK_2,TEKN_DEKK_3,TEKN_DEKK_F,TEKN_FELG_1,TEKN_FELG_2,TEKN_FELG_3,TEKN_MILI_1,TEKN_MILI_2,TEKN_MILI_3,TEKN_HAST_1,TEKN_HAST_2,TEKN_HAST_3,TEKN_INNP_1,TEKN_INNP_2,TEKN_INNP_3,TEKN_SPOR_1,TEKN_SPOR_2,TEKN_SPOR_3,TEKN_LAST_1,TEKN_LAST_2,TEKN_LAST_3,TEKN_LUFT_1,TEKN_LUFT_2,TEKN_LUFT_3,TEKN_NGN,TEKN_EU_HOVED,TEKN_EURONORM_NY,TEKN_EU_TYPE,TEKN_EU_VARIANT,TEKN_EU_VERSJON,TEKN_SISTE_PKK,TEKN_NESTE_PKK,TEKN_STANDSTOY,TEKN_PARTIKKELFILTER,TEKN_NOX_UTSLIPP_MGPRKH,TEKN_NOX_UTSLIPP_GPRKWH,TEKN_PARTIKKELUTSLIPP,TEKN_MAALEMETODE,TEKN_CO2_UTSLIPP,TEKN_DRIVSTOFF_FORBRUK )
// VALUES (:%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d,:%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d,:%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d,:%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d,:%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d,:%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d,:%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d,:%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d,:%d, :%d, :%d, :%d, :%d, :%d )`
//	query4 = `
// INTO ADMIN.EXT_MVR_TEXT_DATA
// VALUES (:%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d,:%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d,:%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d,:%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d,:%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d,:%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d,:%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d,:%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d, :%d,:%d, :%d, :%d, :%d, :%d, :%d )`
//
//	sfmt2 = `into ADMIN.EXT_MVR_TEXT_DATA ( TEKN_KJM,TEKN_KJM_PERSJ,TEKN_KJM_FARGE,TEKN_UNR,TEKN_REG_F_G,TEKN_REG_F_G_N,TEKN_REG_EIERSKIFTE_DATO,TEKN_REG_EIER_DATO,TEKN_AVREG_DATO,TEKN_VRAKET_DATO,TEKN_UTFORT_DATO,TEKN_REG_STATUS,TEKN_BRUKTIMP,TEKN_MERKE,TEKN_MERKENAVN,TEKN_MODELL,TEKN_TKNAVN,TEKN_TUK_VERDI,TEKN_KJTGRP,TEKN_MOTORKODE,TEKN_MOTORYTELSE,TEKN_SLAGVOLUM,TEKN_DRIVST,TEKN_GIRKASSE,TEKN_HYBRID,TEKN_HYBRID_KATEGORI,TEKN_TOTVEKT,TEKN_EGENVEKT,TEKN_VOGNTOGVEKT,TEKN_MAKS_TAKLAST,TEKN_THV_M_BREMS,TEKN_THV_U_BREMS,TEKN_BEL_H_FESTE,TEKN_LENGDE,TEKN_BREDDE,TEKN_ANTALL_DORER,TEKN_FARGE,TEKN_PABYGG,TEKN_SITTEPLASSERFORAN,TEKN_SITTEPLASSER_TOTALT,TEKN_AKSLER,TEKN_AKSLER_DRIFT,TEKN_MINAVST_MS1,TEKN_MINAVST_MS2,TEKN_DEKK_1,TEKN_DEKK_2,TEKN_DEKK_3,TEKN_DEKK_F,TEKN_FELG_1,TEKN_FELG_2,TEKN_FELG_3,TEKN_MILI_1,TEKN_MILI_2,TEKN_MILI_3,TEKN_HAST_1,TEKN_HAST_2,TEKN_HAST_3,TEKN_INNP_1,TEKN_INNP_2,TEKN_INNP_3,TEKN_SPOR_1,TEKN_SPOR_2,TEKN_SPOR_3,TEKN_LAST_1,TEKN_LAST_2,TEKN_LAST_3,TEKN_LUFT_1,TEKN_LUFT_2,TEKN_LUFT_3,TEKN_NGN,TEKN_EU_HOVED,TEKN_EURONORM_NY,TEKN_EU_TYPE,TEKN_EU_VARIANT,TEKN_EU_VERSJON,TEKN_SISTE_PKK,TEKN_NESTE_PKK,TEKN_STANDSTOY,TEKN_PARTIKKELFILTER,TEKN_NOX_UTSLIPP_MGPRKH,TEKN_NOX_UTSLIPP_GPRKWH,TEKN_PARTIKKELUTSLIPP,TEKN_MAALEMETODE,TEKN_CO2_UTSLIPP,TEKN_DRIVSTOFF_FORBRUK )
//VALUES (	'%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s') `
//	sfmt = `
//INSERT INTO ADMIN.EXT_MVR_TEXT_DATA ( "TEKN_KJM","TEKN_KJM_PERSJ","TEKN_KJM_FARGE","TEKN_UNR","TEKN_REG_F_G","TEKN_REG_F_G_N","TEKN_REG_EIERSKIFTE_DATO","TEKN_REG_EIER_DATO","TEKN_AVREG_DATO","TEKN_VRAKET_DATO","TEKN_UTFORT_DATO","TEKN_REG_STATUS","TEKN_BRUKTIMP","TEKN_MERKE","TEKN_MERKENAVN","TEKN_MODELL","TEKN_TKNAVN","TEKN_TUK_VERDI","TEKN_KJTGRP","TEKN_MOTORKODE","TEKN_MOTORYTELSE","TEKN_SLAGVOLUM","TEKN_DRIVST","TEKN_GIRKASSE","TEKN_HYBRID","TEKN_HYBRID_KATEGORI","TEKN_TOTVEKT","TEKN_EGENVEKT","TEKN_VOGNTOGVEKT","TEKN_MAKS_TAKLAST","TEKN_THV_M_BREMS","TEKN_THV_U_BREMS","TEKN_BEL_H_FESTE","TEKN_LENGDE","TEKN_BREDDE","TEKN_ANTALL_DORER","TEKN_FARGE","TEKN_PABYGG","TEKN_SITTEPLASSERFORAN","TEKN_SITTEPLASSER_TOTALT","TEKN_AKSLER","TEKN_AKSLER_DRIFT","TEKN_MINAVST_MS1","TEKN_MINAVST_MS2","TEKN_DEKK_1","TEKN_DEKK_2","TEKN_DEKK_3","TEKN_DEKK_F","TEKN_FELG_1","TEKN_FELG_2","TEKN_FELG_3","TEKN_MILI_1","TEKN_MILI_2","TEKN_MILI_3","TEKN_HAST_1","TEKN_HAST_2","TEKN_HAST_3","TEKN_INNP_1","TEKN_INNP_2","TEKN_INNP_3","TEKN_SPOR_1","TEKN_SPOR_2","TEKN_SPOR_3","TEKN_LAST_1","TEKN_LAST_2","TEKN_LAST_3","TEKN_LUFT_1","TEKN_LUFT_2","TEKN_LUFT_3","TEKN_NGN","TEKN_EU_HOVED","TEKN_EURONORM_NY","TEKN_EU_TYPE","TEKN_EU_VARIANT","TEKN_EU_VERSJON","TEKN_SISTE_PKK","TEKN_NESTE_PKK","TEKN_STANDSTOY","TEKN_PARTIKKELFILTER","TEKN_NOX_UTSLIPP_MGPRKH","TEKN_NOX_UTSLIPP_GPRKWH","TEKN_PARTIKKELUTSLIPP","TEKN_MAALEMETODE","TEKN_CO2_UTSLIPP","TEKN_DRIVSTOFF_FORBRUK" )
//VALUES ('%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s','%s');
//`
//)
