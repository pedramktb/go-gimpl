# Go-Gimpl Architecture Optimization Tasks

## 待办清单

- [ ] **1. 数据切片预分配（Slice Pre-allocation）**
- [ ] **2. 分页查询合并（Combine Pagination Queries）**
- [ ] **3. SQL 字符串构建优化（String Building）**
- [ ] **4. 字段查询缓存（Field Lookup Caching）**
- [ ] **5. JSON AST 树反序列化的“重复解析”开销优化**
- [ ] **6. 游标 (Cursor) 编解码中 CSV 解析器的替换**
- [ ] **7. 运行时高频深度反射的抽象屏障优化**

---

## 详细优化项目参考

### 1. 数据切片预分配（Slice Pre-allocation）⭐ 最高优先级

- **位置**: `pgimp/finder.go:137-140` 中的 `fetch()` 函数
- **当前问题**: `append()` 在循环中无预分配，导致处理大结果集时多次触发切片容量扩展、内存分配和数据复制。
- **优化方案**: 从 `Find()` 传递已知结果数量，或在循环前 `make([]E, 0, estimatedCapacity)` 预分配估计容量。
- **影响/ROI**: 🔴 高影响。大数据集性能提升 20-50%，内存分配降至 O(1)。实现难度低。ROI: ⭐⭐⭐⭐⭐

### 2. 分页查询合并（Combine Pagination Queries）⭐ 第二优先级

- **位置**: `pgimpl/finder.go:55-80` 中的 `Find()` 方法
- **当前问题**: 执行 3 个独立的数据库查询（Count, Main, Prev），产生冗余的网络往返和数据库查询压力。
- **优化方案**: 使用 CTE（Common Table Expressions）或窗口函数将 3 个查询合并为 1 个带有 `UNION ALL` 的 SQL 请求。
- **影响/ROI**: 🔴 高影响。网络往返减少 66%，DB CPU 降低 30-40%。实现难度中等。ROI: ⭐⭐⭐⭐

### 3. SQL 字符串构建优化（String Building）⭐ 第三优先级

- **位置**: `pgimpl/expr.go` 树遍历生成 SQL 的相关方法 (`builder.build`, `jsonExpr`, `buildColumnQuant` 等)。
- **当前问题**: 广泛使用 `+` 拼接字符串产生大量临时对象，触发频繁内存分配。
- **优化方案**: 引入并在上下文共享 `strings.Builder`，使用流式写入替代字符串直接拼接。
- **影响/ROI**: 🟡 中等影响。内存分配减少 70-80%，复杂查询性能提升。实现难度低。ROI: ⭐⭐⭐

### 4. 字段查询缓存（Field Lookup Caching）⭐ 第四优先级

- **位置**: `pgimpl/pagination.go:25-40` 中的 `cursorQuery()` 函数
- **当前问题**: 嵌套循环中重复调用 `fieldColumn()` 查询相同字段列名，产生 O(n²) 级别的冗余查询。
- **优化方案**: 在循环外构建一个 `map[string]string` 作为本地缓存，后续循环直接读取缓存。
- **影响/ROI**: 🟡 中等影响。有效减少重复方法调用，改善多字段排序性能。实现难度低。ROI: ⭐⭐⭐

### 5. JSON AST 树反序列化的“重复解析”开销优化

- **位置**: `gimpl/expr.go` 中的 `decodeExpr()` 
- **当前问题**: 识别 Op 操作符时先序列化一次 `probe`，随后对相同字节流再次 `Unmarshal(src, &tmp)`。
- **优化方案**: 使用带全量字段的结构体或 `json.Decoder` Token 流式解析，避免一次处理两次全量反序列化。
- **影响**: 降低 CPU 损耗与堆内存 GC 压力。

### 6. 游标 (Cursor) 编解码中 CSV 解析器的替换

- **位置**: `gimpl/pagination.go` 中的 `FromStr()`
- **当前问题**: 采用厚重的标准库 `csv.NewReader` 及内存 `[]byte` 规制替换来提取 Cursor 数据。
- **优化方案**: 直接使用无状态 JSON 数组 (加 Base64) 或轻量的字符串分割代替。
- **影响**: 减少高频翻页查询的 CPU 负担和内存拷贝开销。

### 7. 运行时高频深度反射的抽象屏障优化

- **位置**: `gimpl/expr.go` (`resolveVal`, `resolveArrayElem`) 及 `gimpl/pagination.go` (`Sorts.Cursor()`)
- **当前问题**: 数据推断中强依赖 `reflect.TypeOf()`, `reflect.New()` 等堆栈层面的反射。
- **优化方案**: 结合 `sync.Map` 搭建类型缓存池，或运用 Go 1.18+ 泛型。
- **影响**: 减少运行时动态检查产生的 Reflection Barrier 损耗。

---
**测试策略**: 所有优化实现后均应补充/运行完整的单元测试、基准测试(Benchmark)和集成测试，确保其作为底层数据访问组件的绝对稳定性。
