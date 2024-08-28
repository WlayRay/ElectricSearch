package kvdb

import (
	"errors"
	"sync/atomic"

	bolt "go.etcd.io/bbolt"
)

var ErrNoData = errors.New("没有数据")

// Bolt 存储结构
type Bolt struct {
	db     *bolt.DB
	path   string
	bucket []byte
}

// 使用 Builder 模式来构建 Bolt 结构体
func (s *Bolt) WithDataPath(path string) *Bolt {
	s.path = path
	return s
}

func (s *Bolt) WithBucket(bucket string) *Bolt {
	s.bucket = []byte(bucket)
	return s
}

// 初始化DB
func (s *Bolt) Open() error {
	dataDir := s.GetDbPath()
	db, err := bolt.Open(dataDir, 0o600, bolt.DefaultOptions)
	if err != nil {
		return err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(s.bucket)
		return err
	})
	if err != nil {
		db.Close()
		return err
	} else {
		s.db = db
		return nil
	}
}

// 获取存储数据的目录
func (s *Bolt) GetDbPath() string {
	return s.path
}

// WALName returns the path to currently open database file.(额外定义的方法）
func (s *Bolt) WALName() string {
	return s.db.Path()
}

// 写入<key, value>
func (s *Bolt) Set(k, v []byte) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(s.bucket).Put(k, v)
	})
}

// 批量写入<key, value>
func (s *Bolt) BatchSet(keys, values [][]byte) error {
	if len(keys) != len(values) {
		return errors.New("the key and the value are not the same length")
	}
	s.db.Batch(func(tx *bolt.Tx) error {
		for i, key := range keys {
			value := values[i]
			if err := tx.Bucket(s.bucket).Put(key, value); err != nil {
				return err
			}
		}
		return nil
	})
	return nil
}

// 读取key对应的value
func (s *Bolt) Get(k []byte) ([]byte, error) {
	var ival []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		ival = tx.Bucket(s.bucket).Get(k)
		return nil
	})
	if len(ival) == 0 {
		return nil, ErrNoData
	}
	return ival, err
}

// 批量读取，注意不保证顺序
func (s *Bolt) BatchGet(keys [][]byte) ([][]byte, error) {
	values := make([][]byte, len(keys))
	s.db.Batch(func(tx *bolt.Tx) error {
		for i, key := range keys {
			ival := tx.Bucket(s.bucket).Get(key)
			values[i] = ival
		}
		return nil
	})
	return values, nil
}

// 删除
func (s *Bolt) Delete(k []byte) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(s.bucket).Delete(k)
	})
}

// 批量删除
func (s *Bolt) BatchDelete(keys [][]byte) error {
	s.db.Batch(func(tx *bolt.Tx) error {
		for _, key := range keys {
			if err := tx.Bucket(s.bucket).Delete(key); err != nil {
				return err
			}
		}
		return nil
	})
	return nil
}

// 判断某个key是否存在
func (s *Bolt) Has(k []byte) bool {
	var b []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		b = tx.Bucket(s.bucket).Get(k)
		return nil
	})

	if err != nil || string(b) == "" {
		return false
	}

	return true
}

// 遍历数据库，返回数据的条数
func (s *Bolt) IterDB(fn func(k, v []byte) error) int64 {
	var total int64
	s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bucket)
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if err := fn(k, v); err != nil {
				return err
			} else {
				atomic.AddInt64(&total, 1)
			}
		}
		return nil
	})

	return atomic.LoadInt64(&total)
}

// 遍历所有key，返回数据的条数
func (s *Bolt) IterKey(fn func(k []byte) error) int64 {
	var total int64
	s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(s.bucket)
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			if err := fn(k); err != nil {
				return err
			} else {
				atomic.AddInt64(&total, 1)
			}
		}
		return nil
	})
	return atomic.LoadInt64(&total)
}

// 释放所有数据库资源。在关闭数据库之前，必须先关闭所有事务。
func (s *Bolt) Close() error {
	return s.db.Close()
}
