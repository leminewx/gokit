// Package line 提供折线与线段抽象，以及点在线段、折线上的最近点查询。
//
// 距离口径通过注入 geography.DistanceCalculator 决定（Haversine 或 Vincenty），
// 因此同一份顶点数据可以在球面与椭球模型间切换，结果随注入函数变化。
//
// 构造时校验顶点数量与基准面一致性，非法输入返回对应哨兵错误。
package line
