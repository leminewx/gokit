// Package projection 提供经纬度到平面坐标的投影（Web Mercator、Lambert 等积圆柱、
// 以指定点为原点的本地投影），以及平面点集的距离、面积、质心与线段相交运算。
//
// 平面点统一使用 Point（[2]float64，x/y）；投影器通过 Projector 接口注入，
// 需要局部坐标时用 NewLocalProjector 包一层基准投影器。纬度越界由 ValidateLatitude 判定。
//
// 投影会引入形变，距离与面积运算只在投影平面内自洽；跨大范围测量应回到 geography。
package projection
