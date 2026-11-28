// Package knn 提供 K 近邻分类模型 SimpleModel。
//
// 模型用训练样本与标签做惰性预测：Predict 取距离最近的 k 个样本多数投票，距离度量可
// 通过 SetDistanceCalculator 替换（默认欧氏距离）。未训练、k 非法或样本维度不一致时
// 返回哨兵错误；训练完成后模型只读，可并发预测。
package knn
