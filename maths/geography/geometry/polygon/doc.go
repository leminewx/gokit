// Package polygon 提供多边形构造与射线法包含判断。
//
// 多边形顶点按顺序闭合，面积与周长使用注入的 geography.DistanceCalculator 计算；
// IsPointInPolygonByRay 的 onEdge 参数决定边界点是否算作内部。
//
// 本包只做平面/球面近似判断，不处理自相交多边形的奇偶规则歧义。
package polygon
