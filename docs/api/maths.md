# maths 数学 API

导入前缀：`gitee.com/leminewx/gokit/maths/...`

## maths（统计函数）

所有函数都以根包 `gokit.Number` 为类型约束，可混用整型与浮点型。

| 函数 | 说明 |
|------|------|
| `Sum[T](values ...T) (sum T)` | 求和 |
| `CalculateMean[T](values ...T) float64` | 算术平均 |
| `CalculateVariance[T](values []T, mean ...float64) float64` | 总体方差，可传入已知均值 |
| `CalculateStandardDeviation[T](values []T, mean ...float64) float64` | 总体标准差 |
| `CalculateMedian[T](values []T, sorted ...bool) float64` | 中位数，可声明已排序 |
| `Count[T](values ...T) map[T]int` | 频次统计 |
| `Range[T](values ...T) (min, max T)` | 极值 |
| `CalculateAbs[T](value T) T` | 绝对值 |

```go
mean := maths.CalculateMean(1, 2, 3, 4)        // 2.5
sd := maths.CalculateStandardDeviation([]float64{1, 2, 3}, mean)
```

## maths/percentile（百分位）

```go
p := percentile.NewExact()
p.Add(1.2) // 单个数据点
p.Add(3.4)
p.Add(5.6)
v, err := p.CalculatePercentile(95) // P95

d := percentile.NewTDigest(100) // 压缩因子，典型 100
d.Add(12.5)
p99, _ := d.CalculatePercentile(99)
_ = d.Merge(p) // 仅同类型可合并，否则返回 error

n := percentile.NewNearestRank() // 最近秩：结果必定是真实样本点
n.Add(12)
n.Add(30)
n.Add(58)
p50, _ := n.CalculatePercentile(50)

// 已排序切片可直接计算，免去实例开销
if v, err := percentile.CalculateNearestRank([]float64{5, 10, 15}, 90); err == nil {
    _ = v
}

// 泛型入口：数据点按调用方类型添加，计算域仍为 float64
t := percentile.NewTypedNearestRank[int64]()
t.Add(120_000)
p99, _ := t.CalculatePercentile(99)
```

| 符号 | 说明 |
|------|------|
| `Percentiler` 接口 | `Add(float64)`、`GetCount() int`、`Merge(Percentiler) error`、`CalculatePercentile(p float64) (float64, error)` |
| `NewExact() Percentiler` | 保留全部样本，R-7 线性插值（与 Excel `PERCENTILE.INC` 一致） |
| `NewTDigest(compression float64) Percentiler` | T-Digest 近似，内存恒定，尾部精度更高 |
| `NewNearestRank() Percentiler` | 标准最近秩法：下标 `ceil(n*p/100)-1`，结果为真实观测值，不做插值 |
| `CalculateNearestRank(sorted []float64, p float64) (float64, error)` | 在已排序切片上直接算最近秩，不拷贝不排序 |
| `TypedPercentiler[T gokit.Number]` | 泛型输入包装：`Add(T)`、`GetCount()`、`Merge(*TypedPercentiler[T])`、`CalculatePercentile(p float64) (float64, error)` |
| `NewTypedExact[T]()` / `NewTypedTDigest[T](compression)` / `NewTypedNearestRank[T]()` | 分别包装三种口径，输入类型为 `T`；`Merge` 仅接受同算法同类型的实例 |
| `GetPercentiler() Percentiler` | 取回底层非泛型实例，与既有 API 互操作 |

约束：至少一个数据点，`p ∈ [0,100]`，越界或空数据返回 `error`；三种实现不可互相合并。

泛型入口的精度边界：内部按 float64 计算，可精确表示的整数上限为 2^53。超过该范围的
`int64` / `uint64` 数据点在 `Add` 时即被舍入（如 2^53+1 变为 2^53），分位结果无法还原原值；
需要完整保留整数精度时，应选择返回样本原值的最近秩口径，而不是 `TypedPercentiler`。

口径差异：`NewExact`（R-7）与 `NewNearestRank`（最近秩）对同一批样本会给出不同数值
（如 4 个样本的 P50 分别为 2.5 与 2），监控/报表选定口径后不要混用。

## maths/geography（地理坐标）

三种坐标：`LLAPoint`（经度、纬度、海拔）、`ECEFPoint`（地心直角 X/Y/Z）、
`ENUPoint`（相对参考点的东/北/天偏移）。构造参数顺序与 `GetValues` 返回顺序一致。

```go
p1 := geography.NewLLAPoint(datum.WGS84, 113.9, 22.5, 10) // 经度, 纬度, 海拔
p2 := geography.NewLLAPoint(datum.WGS84, 113.91, 22.51, 12)

dist, az1, az2, err := geography.CalculateHaversineDistance(p1, p2)
e := p1.ConvertToECEFPoint()
enu := p1.ConvertToENUPoint(p2) // 相对 p2 的东北天坐标
```

| 符号 | 说明 |
|------|------|
| `NewLLAPoint(dat datum.Datum, values ...float64) LLAPoint` | 三个分量缺省补 0 |
| `NewLLAPoints(dat, points ...[3]float64) []LLAPoint` | 批量构造 |
| `NewECEFPoint` / `NewECEFPoints` | ECEF 构造 |
| `NewENUPoint` / `NewENUPoints` | ENU 构造 |
| `(LLAPoint) GetValues() (lon, lat, alt float64)` / `GetRadianValues()` | 十进制度 / 弧度 |
| `(LLAPoint) ConvertToECEFPoint() ECEFPoint` / `ConvertToENUPoint(refLLA LLAPoint) ENUPoint` | 坐标转换 |
| `(ECEFPoint) ConvertToENUPoint(refLLA LLAPoint) ENUPoint` | ECEF → ENU |
| `GetDatum()` / `IsEqualTo()` / `String()` | 基准面、相等、文本 |
| `CalculateHaversineDistance(p1, p2) (distance, startAzimuth, endAzimuth float64, err error)` | 球面距离（快） |
| `CalculateVincentyDistance(p1, p2) (distance, startAzimuth, endAzimuth float64, err error)` | 椭球距离（精确） |
| `CalculatePolygonHaversinePerimeter(points ...LLAPoint) (float64, error)` | 多边形周长（球面） |
| `CalculatePolygonVincentyPerimeter(points ...LLAPoint) (float64, error)` | 多边形周长（椭球） |
| `ConvertToRadian(value float64) float64` | 角度转弧度 |
| `Deduplicate[T Equalter[T]](points ...T) []T` | 去重 |
| `BASIC_EPSILON` | `1e-7`，坐标比较容差 |
| `ErrDatumNotMatched` | 基准面不一致错误 |
| `DistanceCalculator` / `PerimeterCalculator` | 距离 / 周长函数类型，便于注入 |

### maths/geography/datum

用位标志组合基准面：`Datum` 由 `CRS`（坐标参考系）与 `Ellipsoid`（椭球体）按位或得到。

```go
p := geography.NewLLAPoint(datum.WGS84, 113.9, 22.5, 0)
```

| 符号 | 说明 |
|------|------|
| `Datum uint16` | 基准面（`CRS \| Ellipsoid`） |
| `CRS uint16` / `Ellipsoid uint16` | 参考系 / 椭球体位标志 |
| `WGS84` | `Datum(CRS_WGS84 \| ELLIPSOID_WGS84)` |
| `CRS_WGS84` / `ELLIPSOID_WGS84` | WGS84 对应位 |

### maths/geography/geometry

GeoJSON 风格的几何数据结构，均实现 `Geometry` 接口：

`Point`、`MultiPoint`、`LineString`、`MultiLineString`、`Polygon`、`MultiPolygon`、
`GeometryCollection`，以及 `Feature`、`FeatureCollection`。

### maths/geography/geometry/line

| 符号 | 说明 |
|------|------|
| `NewLLAPolyline(points []geography.LLAPoint, calc geography.DistanceCalculator) (Polyline, error)` | 折线 |
| `NewLLASegment(start, end geography.LLAPoint, calc geography.DistanceCalculator) (Segment, error)` | 线段 |
| `NewLLASegments(points []geography.LLAPoint, calc geography.DistanceCalculator) ([]Segment, error)` | 折线拆分为线段 |
| `FindNearestPointOnPolyline(polyline Polyline, target projection.Point) (minDistance float64, segmentIdx int)` | 最近点与所在线段下标 |
| `Polyline` / `Segment` 接口 | 折线 / 线段抽象 |

### maths/geography/geometry/polygon

```go
poly, err := polygon.NewLLAPolygon(vertexes, geography.CalculateHaversineDistance)
inside := polygon.IsPointInPolygonByRay(poly, target, true) // onEdge = true 表示边界算内部
```

| 符号 | 说明 |
|------|------|
| `NewLLAPolygon(vertexes []geography.LLAPoint, calc geography.DistanceCalculator) (Polygon, error)` | 构造多边形 |
| `IsPointInPolygonByRay(p Polygon, target projection.Point, onEdge ...bool) bool` | 射线法包含判断 |
| `Polygon` / `LLAPolygon` | 多边形接口与实现 |

### maths/geography/projection

平面投影与平面几何运算。`Point` 为 `[2]float64`（x, y）。

```go
mercator := projection.NewWebMercatorProjector(datum.WGS84)
local := projection.NewLocalProjector(mercator, center) // 以 center 为原点的局部坐标

area, centroid, err := projection.CalculatePolygonAreaAndCentroid(points...)
```

| 符号 | 说明 |
|------|------|
| `NewWebMercatorProjector(dat datum.Datum) Projector` | Web Mercator（EPSG:3857） |
| `NewLambertCylindricalEqualAreaProjector(dat datum.Datum) Projector` | Lambert 等积圆柱 |
| `NewLocalProjector(base Projector, center geography.LLAPoint) Projector` | 以某点为中心的局部平面 |
| `CalculateEuclidDistance(points ...Point) float64` | 折线长度 |
| `CalculatePointToLineDistance(segment [2]Point, target Point) float64` | 点到直线距离 |
| `CalculatePointToSegmentDistance(segment [2]Point, target Point) float64` | 点到线段距离 |
| `CalculatePolygonArea(points ...Point) float64` | 多边形面积 |
| `CalculatePolygonAreaAndCentroid(points ...Point) (area float64, centroid Point, err error)` | 面积与质心 |
| `IntersectSegments(segmentA, segmentB [2]Point) ([]Point, bool)` | 线段相交点 |
| `IsLeftOnSegment` / `IsOnSegment` / `IsOnPolyline` | 位置判断 |
| `Deduplicate(points ...Point) []Point` | 去重 |
| `Code` / `EPSG_3857` | 投影代码（如 `"EPSG:3857"`） |
| `MIN_POLYGON_AREA` | `1e-10` |

## maths/matrix/vector

矩阵能力目前以向量子包提供（导入路径 `maths/matrix/vector`）。

```go
a := vector.Vector2D[float64]{3, 4}
b := vector.Vector2D[float64]{1, 2}

dot, err := a.CalculateDot(&b)
sum, err := a.Add(&b)
norm := a.GetNorm() // 5
shape := a.GetShape() // [2]
```

| 符号 | 说明 |
|------|------|
| `Vector[T gokit.Number]` 接口 | `CalculateDot`、`Add`、`Subtract`、`CalculateCross`、`SetValueAt`、`GetValueAt`、`GetNorm`、`GetShape`、`IsSameShapeWith` |
| `Vector1D[T]` / `Vector2D[T]` / `Vector3D[T]` | 定长向量（数组类型） |
| `CalculateWedge[T](a, b []T) ([][]T, bool)` | 楔积（外积） |
| `ErrShapeNotMatch` | 形状不一致 |
| `ErrOneDimensionalDotMeaningless` | 一维叉积无意义 |

## maths/biology/activator

| 符号 | 说明 |
|------|------|
| `Activator` 接口 | 激活函数抽象 |
| `ReLU{}` | 零值可用 |
| `NewLeakyReLU(alpha float64) *LeakyReLU` | 带泄漏斜率 |
| `NewSigmoid() *Sigmoid` / `NewTanh() *Tanh` | S 型 / 双曲正切 |
| `CalculateSoftmax(x []float64) []float64` | 多分类归一化 |
| `CalculateSoftmaxBackward(probes []float64, label int) []float64` | Softmax 反向梯度 |
| `ActivateBatch(activator Activator, inputs []float64) []float64` | 批量激活 |
| `Name` / `NAME_RELU` 等 | 激活函数名称常量 |

`CalculateSoftmax` 对空输入返回 `nil`；其余激活函数均为零值可用。

## maths/learning 与 knn

| 符号 | 说明 |
|------|------|
| `learning.CalculateEuclideanDistance[T](a, b []T) float64` | 欧氏距离 |
| `learning.DistanceCalculator[T]` | 距离函数类型，可替换度量 |
| `knn.NewSimpleModel[T](k uint) *SimpleModel[T]` | K-近邻模型 |
| `(*SimpleModel) Train(samples [][]T, labels []int) error` | 训练 |
| `(*SimpleModel) Predict(object []T) (int, error)` | 预测 |
| `(*SimpleModel) IsTrained() bool` | 是否已训练 |
| `(*SimpleModel) SetDistanceCalculator(learning.DistanceCalculator[T])` | 替换距离度量 |

```go
model := knn.NewSimpleModel[float64](3)
if err := model.Train(samples, labels); err != nil {
    return err
}
label, err := model.Predict(query)
```

## maths/root（求根）

统一返回 `Solution` / `ComplexSolution`，不返回裸元组；收敛失败返回 `ErrNotConverged`，
迭代中出现 `NaN` / `Inf` 立即返回 `ErrNonFinite`。

| 符号 | 说明 |
|------|------|
| `Solution` | `Root`（近似根）、`Value`（根处残差）、`Iterations`（迭代次数） |
| `ComplexSolution` | 同上，`Root` / `Value` 为 `complex128` |
| `Function` / `Derivative` / `ComplexFunction` | `func(float64) float64` / 导数 / `func(complex128) complex128` |
| `CalculateBisection(f, a, b, opts...)` | 二分法：只需端点变号，线性收敛；端点即根时 `Iterations == 0` |
| `CalculateNewton(f, initial, opts...)` | 牛顿法：二次收敛，`WithDerivative` 缺省时用中心差分 |
| `CalculateSecant(f, a, b, opts...)` | 弦截法：无需导数，收敛阶约 1.618 |
| `CalculateFixedPoint(g, initial, opts...)` | 求解 `x = g(x)`；`WithAitken()` 启用 Δ² 加速 |
| `CalculateMuller(f, z0, z1, z2, opts...)` | 抛物线插值，三个初值须互不相同，可求复根 |
| `WithTolerance` / `WithResidual` / `WithMaxIterations` | 自变量容差（默认 `1e-9`）/ 残差容差 / 迭代上限（默认 100） |

```go
solution, err := root.CalculateNewton(func(x float64) float64 { return x*x - 2 }, 1.5)
if err == nil {
    fmt.Println(solution.Root)       // 1.4142135623730951
    fmt.Println(solution.Iterations)
}

complexRoot, err := root.CalculateMuller(func(z complex128) complex128 { return z*z + 1 }, 0, 1, 2)
// complexRoot.Root ≈ ±i
```

## maths/quadrature（数值积分）

`Result.ErrorEstimate` 是算法内部的收敛指示，不是严格误差上界；`0` 表示该方法默认不提供估计。

| 符号 | 说明 |
|------|------|
| `Result` | `Value`（积分近似值）、`Iterations`（Romberg 行数 / 自适应叶子数 / 其余为 1）、`ErrorEstimate` |
| `CalculateRomberg(f, a, b, opts...)` | 梯形序列 + Richardson 外推，自带误差估计 |
| `CalculateCompositeCotes(f, a, b, opts...)` | 复化 Newton-Cotes，`WithPanels` 分区数、`WithOrder(2~5)` 节点数 |
| `CalculateAdaptiveCotes(f, a, b, opts...)` | 按局部误差递归细分，`WithMaxIterations` 为递归深度上限 |
| `CalculateGaussLegendre(f, a, b, opts...)` | 2~5 点高斯节点，`WithOrder` 即节点数，n 点对 2n-1 次多项式精确 |
| `WithTolerance` / `WithErrorEstimation` | 收敛容差；为复化 Cotes 与 Gauss 额外计算估计（约两倍被积函数求值） |

```go
result, err := quadrature.CalculateRomberg(func(x float64) float64 { return math.Sin(x) }, 0, math.Pi)
if err == nil {
    fmt.Println(result.Value, result.ErrorEstimate) // ≈ 2
}
```

区间要求 `a < b` 且两端有限，非法区间返回 `ErrInvalidInterval`。

## maths/interpolator（插值）

越界默认拒绝（返回 `ErrOutOfRange`），需外推时显式传入 `WithExtrapolation`。

| 符号 | 说明 |
|------|------|
| `Polynomial` | `Calculate` / `CalculateDerivative` / `GetDegree` / `GetCoefficients`（返回副本） |
| `NewLagrange(xs, ys)` / `NewNewton(xs, ys)` | 全局多项式插值，样本上限 64；节点须严格递增 |
| `NewCubicSpline(xs, ys, condition, opts...)` | 三次样条（弯矩法），返回 `*CubicSpline` |
| `BOUNDARY_NATURAL` / `CLAMPED` / `PERIODIC` / `NOT_A_KNOT` | 自然 / 夹持 / 周期 / 非扭结边界 |
| `WithClampedDerivatives(left, right)` | 夹持边界的端点一阶导数，缺失返回 `ErrMissingDerivative` |
| `Extrapolation` 与 `EXTRAPOLATION_*` | `REJECT`（默认）/ `LINEAR` / `CUBIC` / `CLAMP` / `WRAP` |
| `NewHermite(xs, ys, derivatives)` | 分段三次 Hermite，节点处函数值与导数均精确 |
| `CalculateDerivative` / `CalculateSecondDerivative` | 样条的一阶 / 二阶导数，同样遵循越界策略 |

```go
spline, err := interpolator.NewCubicSpline(xs, ys, interpolator.BOUNDARY_NATURAL)
if err == nil {
    value, err := spline.Calculate(0.5)
    slope, _ := spline.CalculateDerivative(0.5)
}

// 需要外推时显式开启，误差随距离快速放大
loose, err := interpolator.NewCubicSpline(xs, ys, interpolator.BOUNDARY_NATURAL,
    interpolator.WithExtrapolation(interpolator.EXTRAPOLATION_LINEAR))
```

约束：周期边界要求首末 `y` 在 `1e-10` 内相等且至少 4 个节点；`EXTRAPOLATION_WRAP` 仅对
`BOUNDARY_PERIODIC` 生效，其余组合返回 `ErrInvalidBoundary`。

## maths/ode（常微分方程）

右端函数返回切片的长度必须与状态维度一致，否则返回 `ErrStateLengthChanged`；
入参切片不会被改写，`Trajectory.Ys` 各行互不共享底层数组。

| 符号 | 说明 |
|------|------|
| `Trajectory` | `Xs`（自变量序列）、`Ys`（对应状态）、`Steps`（接受步数）、`Rejected`（拒绝重试次数） |
| `Function` | `func(x float64, y []float64) []float64`，一阶方程组需先降阶 |
| `CalculateRungeKutta4(f, initial, start, stop, step)` | 固定步长 RK4，末步自动截断到 `stop` |
| `CalculateRungeKutta4Step(f, x, state, step)` | 单步推进，返回新状态 |
| `CalculateFehlberg(f, initial, start, stop, opts...)` | RKF45 自适应：四/五阶嵌入对，按局部误差伸缩步长 |
| `WithTolerance` / `WithInitialStep` | 局部误差容差（默认 `1e-8`）/ 初始步长（默认跨度 1/100） |
| `WithStepRange(minimum, maximum)` / `WithMaxIterations` | 步长上下界 / 步数上限（含被拒尝试） |

```go
trajectory, err := ode.CalculateFehlberg(
    func(x float64, y []float64) []float64 { return []float64{y[0]} },
    []float64{1}, 0, 1, ode.WithTolerance(1e-10))
if err == nil {
    last := trajectory.Ys[len(trajectory.Ys)-1][0] // ≈ e
}
```

RKF45 的局部误差估计只用于步长控制，不随结果返回；`Rejected` 可用于判断初始步长是否偏大。
达到 `WithMaxIterations` 仍未走完区间时返回 `ErrNotConverged`。

## maths/differentiator（数值微分）

`Derivative.ErrorEstimate` 是两次估计之差（收敛指示），不是严格误差上界；`0` 表示该路径不提供。

| 符号 | 说明 |
|------|------|
| `Derivative` | `Value`（导数值）、`ErrorEstimate`（误差指示）、`Window`（参与计算的样本数或模板宽度） |
| `Series` | `Values`（平滑值）、`Derivatives`（一阶导数）、`Window`、`Order` |
| `Function` | `func(x float64) float64` |
| `NewSampler(xs, ys)` | 非等距三点公式：点不少于 3 个、横坐标严格递增且有限 |
| `(*Sampler) CalculateDerivative(x)` / `CalculateDerivativeOrder(x, order)` | 一阶 / 指定阶（1 或 2）；两侧模板都可用时取平均 |
| `CalculateFunctionDerivative(f, x)` / `CalculateFunctionSecondDerivative(f, x)` | 五点模板 + 一次 Richardson 外推，精度约六阶 |
| `CalculateSavitzkyGolay(values, window, order)` | 等距样本局部多项式拟合，一次给出平滑值与一阶导数 |

```go
d, err := differentiator.CalculateFunctionDerivative(func(x float64) float64 { return math.Sin(x) }, 0)
// d.Value ≈ 1

series, err := differentiator.CalculateSavitzkyGolay(values, 7, 2)
// series.Values 为平滑值；series.Derivatives 按单位间隔给出，实际间隔 dx 需自行除以 dx
```

约束：Savitzky-Golay 窗口为 3~31 的奇数且大于多项式阶数，阶数上限 6；平滑结果对不超过 `order`
次的多项式精确；两端不足半窗的点保持原值、导数置 0；采样序列求值点越界返回 `ErrOutOfRange`。
含噪数据应先用 `maths/filter` 或直接用 Savitzky-Golay，而非裸差分。

## maths/filter（序列滤波）

| 符号 | 说明 |
|------|------|
| `Series` | `Values`（滤波后序列，长度与输入一致）、`Replaced`（被修正样本数，仅 Hampel 统计） |
| `CalculateMovingAverage(values, window)` | 滑动平均，两端按可用样本收缩窗口 |
| `CalculateMedian(values, window)` | 滑动中值，抑制脉冲噪声，两端保持原值 |
| `CalculateHampel(values, window, threshold)` | 中位数 + `1.4826×MAD` 尺度，偏离超过 `threshold` 倍尺度即替换 |
| `CalculateExponential(values, alpha)` | 一阶指数平滑（因果低通），`alpha ∈ (0,1]` |

```go
series, err := filter.CalculateHampel(values, 7, 3)
// series.Values 为剔除离群点后的序列
fmt.Println(series.Replaced)

smoothed, err := filter.CalculateExponential(values, 0.2) // 首个样本作为初始状态
```

约束：窗口为不小于 3 的奇数且不超过序列长度，阈值必须为正。指数平滑存在相位滞后，
需要零相位对齐时应在正反两个方向各做一次。Hampel 在 `MAD == 0` 时把任何偏离中位数的样本都视为离群。

## maths/fitter（加权最小二乘拟合）

| 符号 | 说明 |
|------|------|
| `Sample` | `X`、`Y`、`Weight`（非负，0 表示忽略该样本） |
| `Fit` | `Coefficients`（常数项到 `degree` 次项）、`WeightedResidual`、`WeightedTotal`、`RSquared`、`Samples` |
| `CalculatePolynomial(samples, degree)` | 加权多项式拟合，阶数上限 12 |
| `(*Fit) CalculateValue(x)` | Horner 法求拟合值 |

```go
fit, err := fitter.CalculatePolynomial([]fitter.Sample{
    {X: 0, Y: 1, Weight: 1},
    {X: 1, Y: 2, Weight: 1},
    {X: 2, Y: 5, Weight: 1},
}, 2)
if err == nil {
    fmt.Println(fit.Coefficients, fit.RSquared, fit.CalculateValue(0.5))
}
```

内部用 Householder QR 求解加权范德蒙德方程（避免法方程平方条件数）；列退化返回 `ErrSingularSystem`，
正权重样本数少于 `degree+1` 返回 `ErrTooFewSamples`；加权总平方和为 0 时 `RSquared` 取 1。
入参切片不会被改写。

## maths/optimizer（极小化）

| 符号 | 说明 |
|------|------|
| `Solution` | `X`、`Value`、`Iterations`、`Evaluations` |
| `MultiSolution` | 同上，`X` 为 `[]float64` |
| `Function` / `MultiFunction` | 一元 / 多元目标函数 |
| `MinimizeBrent(f, lower, upper, opts...)` | 抛物线插值 + 黄金分割，只需区间含极小点 |
| `MinimizeNelderMead(f, initial, opts...)` | 无导数单纯形搜索，维度宜不超过 10 |
| `WithTolerance` / `WithMaxIterations` | 收敛容差（默认 `1e-8`）/ 迭代上限（默认 1000） |

```go
solution, err := optimizer.MinimizeBrent(func(x float64) float64 { return (x - 2) * (x - 2) }, -5, 5)

multi, err := optimizer.MinimizeNelderMead(func(point []float64) float64 {
    return point[0]*point[0] + point[1]*point[1]
}, []float64{1, 1}, optimizer.WithTolerance(1e-10))
```

只求局部极小，结果依赖搜索区间或初值；迭代上限内未满足收敛条件返回 `ErrNotConverged`，
目标函数返回 `NaN` / `Inf` 立即返回 `ErrNonFinite`。

## maths/distribution（概率分布）

四种分布构造后不可变，可并发调用；`CalculateQuantile` 的概率须在 `(0,1)` 内，否则返回 `ErrInvalidProbability`。

| 符号 | 说明 |
|------|------|
| `NewNormal(mean, stdDev)` | 正态分布；`CalculatePDF` / `CalculateCDF` / `CalculateQuantile` / `GetMean` / `GetVariance` |
| `NewUniform(lower, upper)` | 连续均匀分布，同一组方法 |
| `NewExponential(rate)` | 指数分布（速率参数化），同一组方法 |
| `NewPoisson(mean)` | 泊松分布；`CalculatePMF(k)` / `CalculateCDF(k)` 返回 `float64`，`CalculateQuantile` 返回 `(int, error)` |

```go
normal, err := distribution.NewNormal(0, 1)
if err == nil {
    p, _ := normal.CalculateCDF(1.96) // ≈ 0.975
}

poisson, err := distribution.NewPoisson(3)
cdf := poisson.CalculateCDF(2) // 13e^-3
```

泊松 PMF 在对数空间用 `Lgamma` 计算以防阶乘溢出；`CalculateCDF` 在均值 + 40√均值 + 40 之外直接返回 1；
`CalculateQuantile` 逐项累加，超过项数上限仍未达到目标概率时返回 `ErrNotConverged`。

## maths/spectrum（傅里叶频谱）

| 符号 | 说明 |
|------|------|
| `Spectrum` | `Frequencies`（长度 N/2+1）、`Amplitudes`（已按窗相干增益归一化）、`Resolution`（`sampleRate/N`） |
| `CalculateFFT(values)` | 迭代基 2 Cooley-Tukey，返回复数谱 |
| `CalculateIFFT(values)` | 共轭法反变换，含 `1/N` 归一化 |
| `CalculateWindow(window, length)` | 生成周期窗系数 |
| `CalculateSpectrum(values, sampleRate, window)` | 加窗 + FFT + 单边幅度谱 |
| `Window` 与 `WINDOW_RECTANGULAR` / `WINDOW_HANN` / `WINDOW_HAMMING` / `WINDOW_BLACKMAN` | 窗类型 |

```go
result, err := spectrum.CalculateSpectrum(values, 1000, spectrum.WINDOW_HANN)
if err == nil {
    fmt.Println(result.Frequencies[3], result.Amplitudes[3], result.Resolution)
}
```

约束：输入长度必须为 2 的幂（否则 `ErrInvalidLength`），采样率必须为正，样本中出现 `NaN` / `Inf`
返回 `ErrNonFinite`。窗按 `w[n] = f(2πn/N)` 的周期形式生成，`length == 1` 时返回 `[1]`；
单边幅度谱按窗系数之和归一化，直流与 Nyquist 分量不翻倍。

## maths/sampler（随机采样）

| 符号 | 说明 |
|------|------|
| `Sourcer` 接口 | `GetFloat64()`，须返回 `[0,1)`，越界返回 `ErrInvalidSource` |
| `NewSource(*rand.Rand)` | 把 `math/rand/v2` 生成器包装为 `Sourcer` |
| `NewSampler(source)` | 注入自定义随机源 |
| `NewDefaultSampler(seed)` | 以 PCG 构建固定种子的默认采样器，同种子序列可复现 |
| `(*Sampler) SampleUniform(lower, upper)` | 区间内均匀采样 |
| `(*Sampler) SampleNormal(mean, stdDev)` | Box-Muller 正态采样 |
| `(*Sampler) SampleExponential(rate)` | 逆变换指数采样 |
| `(*Sampler) SamplePoisson(mean)` | 小均值 Knuth 乘积法，`mean ≥ 50` 用正态近似取整 |
| `(*Sampler) ShuffleIndices(count)` | Fisher-Yates 洗牌，返回 `[0, count)` 的排列 |

```go
sampler := sampler.NewDefaultSampler(42) // 固定种子，序列可复现
x, err := sampler.SampleNormal(0, 1)
order, err := sampler.ShuffleIndices(10) // 用于交叉验证、数据集切分

source, err := sampler.NewSource(rand.New(rand.NewPCG(1, 2)))
custom, err := sampler.NewSampler(source) // 需要时换成可复现或加密安全的源
```

`Sampler` 有内部状态，同一实例不可并发调用；需要并发时每 goroutine 各持一个实例。
参数非法分别返回 `ErrInvalidRange` / `ErrInvalidStdDev` / `ErrInvalidRate` / `ErrInvalidMean`，
随机源越过 `[0,1)` 返回 `ErrInvalidSource`，数量非正返回 `ErrInvalidCount`。
