package votingpool_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/p9c/chaincfg/netparams"
	log "github.com/p9c/logi"

	wtxmgr "github.com/p9c/chain/tx/mgr"
	txscript "github.com/p9c/chain/tx/script"
	"github.com/p9c/util"

	waddrmgr "github.com/p9c/wallet/addrmgr"
	walletdb "github.com/p9c/wallet/db"
	_ "github.com/p9c/wallet/db/bdb"
	"github.com/p9c/wallet/votingpool"
)

var (
	pubPassphrase  = []byte("pubPassphrase")
	privPassphrase = []byte("privPassphrase")
	seed           = bytes.Repeat([]byte{0x2a, 0x64, 0xdf, 0x08}, 8)
	fastScrypt     = &waddrmgr.ScryptOptions{N: 16, R: 8, P: 1}
)

func createWaddrmgr(ns walletdb.ReadWriteBucket, params *netparams.Params) (*waddrmgr.Manager, error) {
	err := waddrmgr.Create(ns, seed, pubPassphrase, privPassphrase, params,
		fastScrypt, time.Now())
	if err != nil {
		return nil, err
	}
	return waddrmgr.Open(ns, pubPassphrase, params)
}
func ExampleCreate() {
	// Create a new walletdb.DB. See the walletdb docs for instructions on how
	// to do that.
	db, dbTearDown, err := createWalletDB()
	if err != nil {
		log.L.Error(err)
		return
	}
	defer dbTearDown()
	dbtx, err := db.BeginReadWriteTx()
	if err != nil {
		log.L.Error(err)
		return
	}
	defer func() {
		err := dbtx.Commit()
		if err != nil {
			log.L.Error(err)
		}
	}()
	// Create a new walletdb namespace for the address manager.
	mgrNamespace, err := dbtx.CreateTopLevelBucket([]byte("waddrmgr"))
	if err != nil {
		log.L.Error(err)
		return
	}
	// Create the address manager.
	mgr, err := createWaddrmgr(mgrNamespace, &netparams.MainNetParams)
	if err != nil {
		log.L.Error(err)
		return
	}
	// Create a walletdb namespace for votingpools.
	vpNamespace, err := dbtx.CreateTopLevelBucket([]byte("votingpool"))
	if err != nil {
		log.L.Error(err)
		return
	}
	// Create a voting pool.
	_, err = votingpool.Create(vpNamespace, mgr, []byte{0x00})
	if err != nil {
		log.L.Error(err)
		return
	}
	// Output:
	//
}

// This example demonstrates how to create a voting pool with one
// series and get a deposit address for that series.
func Example_depositAddress() {
	// Create the address manager and votingpool DB namespace. See the example
	// for the Create() function for more info on how this is done.
	teardown, db, mgr := exampleCreateDBAndMgr()
	defer teardown()
	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := votingpoolNamespace(tx)
		// Create the voting pool.
		pool, err := votingpool.Create(ns, mgr, []byte{0x00})
		if err != nil {
			return err
		}
		// Create a 2-of-3 series.
		seriesID := uint32(1)
		requiredSignatures := uint32(2)
		pubKeys := []string{
			"xpub661MyMwAqRbcFDDrR5jY7LqsRioFDwg3cLjc7tML3RRcfYyhXqqgCH5SqMSQdpQ1Xh8EtVwcfm8psD8zXKPcRaCVSY4GCqbb3aMEs27GitE",
			"xpub661MyMwAqRbcGsxyD8hTmJFtpmwoZhy4NBBVxzvFU8tDXD2ME49A6JjQCYgbpSUpHGP1q4S2S1Pxv2EqTjwfERS5pc9Q2yeLkPFzSgRpjs9",
			"xpub661MyMwAqRbcEbc4uYVXvQQpH9L3YuZLZ1gxCmj59yAhNy33vXxbXadmRpx5YZEupNSqWRrR7PqU6duS2FiVCGEiugBEa5zuEAjsyLJjKCh",
		}
		err = pool.CreateSeries(ns, votingpool.CurrentVersion, seriesID, requiredSignatures, pubKeys)
		if err != nil {
			return err
		}
		// Create a deposit address.
		addr, err := pool.DepositScriptAddress(seriesID, votingpool.Branch(0), votingpool.Index(1))
		if err != nil {
			return err
		}
		fmt.Println("Generated deposit address:", addr.EncodeAddress())
		return nil
	})
	if err != nil {
		log.L.Error(err)
		return
	}
	// Output:
	// Generated deposit address: 51pQm3LmtcK6e4rgGoJDpdCw2N4uWZB9wr
}

// This example demonstrates how to empower a series by loading the private
// key for one of the series' public keys.
func Example_empowerSeries() {
	// Create the address manager and votingpool DB namespace. See the example
	// for the Create() function for more info on how this is done.
	teardown, db, mgr := exampleCreateDBAndMgr()
	defer teardown()
	// Create a pool and a series. See the DepositAddress example for more info
	// on how this is done.
	pool, seriesID := exampleCreatePoolAndSeries(db, mgr)
	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := votingpoolNamespace(tx)
		addrmgrNs := addrmgrNamespace(tx)
		// Now empower the series with one of its private keys. Notice that in order
		// to do that we need to unlock the address manager.
		err := mgr.Unlock(addrmgrNs, privPassphrase)
		if err != nil {
			return err
		}
		defer func() {
			err := mgr.Lock()
			if err != nil {
				log.L.Error(err)
			}
		}()
		privKey := "xprv9s21ZrQH143K2j9PK4CXkCu8sgxkpUxCF7p1KVwiV5tdnkeYzJXReUkxz5iB2FUzTXC1L15abCDG4RMxSYT5zhm67uvsnLYxuDhZfoFcB6a"
		return pool.EmpowerSeries(ns, seriesID, privKey)
	})
	if err != nil {
		log.L.Error(err)
		return
	}
	// Output:
	//
}

// This example demonstrates how to use the Pool.StartWithdrawal method.
func Example_startWithdrawal() {
	// Create the address manager and votingpool DB namespace. See the example
	// for the Create() function for more info on how this is done.
	teardown, db, mgr := exampleCreateDBAndMgr()
	defer teardown()
	// Create a pool and a series. See the DepositAddress example for more info
	// on how this is done.
	pool, seriesID := exampleCreatePoolAndSeries(db, mgr)
	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := votingpoolNamespace(tx)
		addrmgrNs := addrmgrNamespace(tx)
		txmgrNs := txmgrNamespace(tx)
		// Create the transaction store for later use.
		txstore := exampleCreateTxStore(txmgrNs)
		// Unlock the manager
		err := mgr.Unlock(addrmgrNs, privPassphrase)
		if err != nil {
			return err
		}
		defer func() {
			err := mgr.Lock()
			if err != nil {
				log.L.Error(err)
			}
		}()
		addr, _ := util.DecodeAddress("1MirQ9bwyQcGVJPwKUgapu5ouK2E2Ey4gX", mgr.ChainParams())
		pkScript, _ := txscript.PayToAddrScript(addr)
		requests := []votingpool.OutputRequest{
			{
				PkScript:    pkScript,
				Address:     addr,
				Amount:      1e6,
				Server:      "server-id",
				Transaction: 123,
			},
		}
		changeStart, err := pool.ChangeAddress(seriesID, votingpool.Index(0))
		if err != nil {
			return err
		}
		// This is only needed because we have not used any deposit addresses from
		// the series, and we cannot create a WithdrawalAddress for an unused
		// branch/idx pair.
		err = pool.EnsureUsedAddr(ns, addrmgrNs, seriesID, votingpool.Branch(1), votingpool.Index(0))
		if err != nil {
			return err
		}
		startAddr, err := pool.WithdrawalAddress(ns, addrmgrNs, seriesID, votingpool.Branch(1), votingpool.Index(0))
		if err != nil {
			return err
		}
		lastSeriesID := seriesID
		dustThreshold := util.Amount(1e4)
		currentBlock := int32(19432)
		roundID := uint32(0)
		_, err = pool.StartWithdrawal(ns, addrmgrNs,
			roundID, requests, *startAddr, lastSeriesID, *changeStart, txstore, txmgrNs, currentBlock,
			dustThreshold)
		return err
	})
	if err != nil {
		log.L.Error(err)
		return
	}
	// Output:
	//
}
func createWalletDB() (walletdb.DB, func(), error) {
	dir, err := ioutil.TempDir("", "votingpool_example")
	if err != nil {
		return nil, nil, err
	}
	db, err := walletdb.Create("bdb", filepath.Join(dir, "wallet.db"))
	if err != nil {
		return nil, nil, err
	}
	dbTearDown := func() {
		db.Close()
		os.RemoveAll(dir)
	}
	return db, dbTearDown, nil
}

var (
	addrmgrNamespaceKey    = []byte("addrmgr")
	txmgrNamespaceKey      = []byte("txmgr")
	votingpoolNamespaceKey = []byte("votingpool")
)

func addrmgrNamespace(dbtx walletdb.ReadWriteTx) walletdb.ReadWriteBucket {
	return dbtx.ReadWriteBucket(addrmgrNamespaceKey)
}
func txmgrNamespace(dbtx walletdb.ReadWriteTx) walletdb.ReadWriteBucket {
	return dbtx.ReadWriteBucket(txmgrNamespaceKey)
}
func votingpoolNamespace(dbtx walletdb.ReadWriteTx) walletdb.ReadWriteBucket {
	return dbtx.ReadWriteBucket(votingpoolNamespaceKey)
}
func exampleCreateDBAndMgr() (teardown func(), db walletdb.DB, mgr *waddrmgr.Manager) {
	db, dbTearDown, err := createWalletDB()
	if err != nil {
		dbTearDown()
		panic(err)
	}
	// Create a new walletdb namespace for the address manager.
	err = walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		addrmgrNs, err := tx.CreateTopLevelBucket(addrmgrNamespaceKey)
		if err != nil {
			return err
		}
		_, err = tx.CreateTopLevelBucket(votingpoolNamespaceKey)
		if err != nil {
			return err
		}
		_, err = tx.CreateTopLevelBucket(txmgrNamespaceKey)
		if err != nil {
			return err
		}
		// Create the address manager
		mgr, err = createWaddrmgr(addrmgrNs, &netparams.MainNetParams)
		return err
	})
	if err != nil {
		dbTearDown()
		panic(err)
	}
	teardown = func() {
		mgr.Close()
		dbTearDown()
	}
	return teardown, db, mgr
}
func exampleCreatePoolAndSeries(db walletdb.DB, mgr *waddrmgr.Manager) (pool *votingpool.Pool, seriesID uint32) {
	err := walletdb.Update(db, func(tx walletdb.ReadWriteTx) error {
		ns := votingpoolNamespace(tx)
		var err error
		pool, err = votingpool.Create(ns, mgr, []byte{0x00})
		if err != nil {
			return err
		}
		// Create a 2-of-3 series.
		seriesID = uint32(1)
		requiredSignatures := uint32(2)
		pubKeys := []string{
			"xpub661MyMwAqRbcFDDrR5jY7LqsRioFDwg3cLjc7tML3RRcfYyhXqqgCH5SqMSQdpQ1Xh8EtVwcfm8psD8zXKPcRaCVSY4GCqbb3aMEs27GitE",
			"xpub661MyMwAqRbcGsxyD8hTmJFtpmwoZhy4NBBVxzvFU8tDXD2ME49A6JjQCYgbpSUpHGP1q4S2S1Pxv2EqTjwfERS5pc9Q2yeLkPFzSgRpjs9",
			"xpub661MyMwAqRbcEbc4uYVXvQQpH9L3YuZLZ1gxCmj59yAhNy33vXxbXadmRpx5YZEupNSqWRrR7PqU6duS2FiVCGEiugBEa5zuEAjsyLJjKCh",
		}
		err = pool.CreateSeries(ns, votingpool.CurrentVersion, seriesID, requiredSignatures, pubKeys)
		if err != nil {
			return err
		}
		return pool.ActivateSeries(ns, seriesID)
	})
	if err != nil {
		panic(err)
	}
	return pool, seriesID
}
func exampleCreateTxStore(ns walletdb.ReadWriteBucket) *wtxmgr.Store {
	err := wtxmgr.Create(ns)
	if err != nil {
		panic(err)
	}
	s, err := wtxmgr.Open(ns, &netparams.MainNetParams)
	if err != nil {
		panic(err)
	}
	return s
}
