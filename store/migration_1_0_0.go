// Copyright 2019 Northern.tech AS
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//        http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package store

import (
	"context"

	"github.com/mendersoftware/go-lib-micro/mongo/migrate"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type migration1_0_0 struct {
	client *mongo.Client
	db     string
}

// Up creates the jobs capped collection
func (m *migration1_0_0) Up(from migrate.Version) error {
	ctx := context.Background()
	m.client.Database(m.db).RunCommand(ctx, bson.M{
		"create": JobsCollectionName,
		"capped": true,
		"size":   1024 * 1024 * 1024 * 64,
	})
	return nil
}

func (m *migration1_0_0) Version() migrate.Version {
	return migrate.MakeVersion(1, 0, 0)
}
