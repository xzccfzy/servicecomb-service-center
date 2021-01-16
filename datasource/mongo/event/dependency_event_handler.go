/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package event

// InstanceEventHandler is the handler to handle events
//as instance registry or instance delete, and notify syncer
//type DependencyEventHandler struct {
//}

//// DependencyEventHandler add or remove the service dependencies
//// when user call find instance api or dependence operation api
//type DependencyEventHandler struct {
//	signals *queue.UniQueue
//}

//func (h *DependencyEventHandler) Type() string {
//	return mongo.CollectionDep
//}
//
//func (h *DependencyEventHandler) eventLoop() {
//	gopool.Go(func(ctx context.Context) {
//		timer := time.NewTimer(1)
//		for {
//			select {
//				case <-ctx.Done():
//					return
//				case <-timer.C:
//					//update dep_rule
//			}
//		}
//	})
//}
//
//func (h *DependencyEventHandler) OnEvent(evt sd.MongoEvent) {
//	return
//}
//
//func (h *DependencyEventHandler) handle() error {
//
//	return nil
//}
//
//func NewDependencyEventHandler() *DependencyEventHandler {
//	return &DependencyEventHandler{}
//}
