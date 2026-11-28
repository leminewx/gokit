// Package learning 提供机器学习基础度量：欧氏距离计算与可替换的 DistanceCalculator
// 函数类型，供 K 近邻等算法注入不同距离口径。
//
// 距离函数对空切片与长度不一致的输入返回 0，不做错误上报；需要严格校验的场景应在
// 调用方先行检查维度。
package learning
