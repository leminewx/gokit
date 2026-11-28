// Package vector 提供定长向量（1D / 2D / 3D）与点积、叉积、模、楔积等运算。
//
// 向量以数组类型实现，元素类型受 gokit.Number 约束；所有运算返回新向量，不修改操作数。
// 形状不一致返回 ErrShapeNotMatch，一维向量的叉积无意义（ErrOneDimensionalDotMeaningless）。
//
// 本包不提供动态长度向量与矩阵分解，矩阵能力以向量运算与外积为基础组合。
package vector
