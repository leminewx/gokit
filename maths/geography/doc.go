// Package geography 提供 LLA（经度、纬度、海拔）、ECEF（地心直角）与 ENU（东北天）
// 三种坐标类型，以及 Haversine / Vincenty 距离与多边形周长计算。
//
// 所有坐标都绑定基准面（datum.Datum），跨基准面的运算返回 ErrDatumNotMatched；
// 角度输入默认按十进制度解释，需要弧度时使用 GetRadianValues。Haversine 适合快速估算，
// Vincenty 在椭球模型下更精确但迭代更慢。
//
// 坐标对象为值类型，构造后不可变；投影与平面几何运算见 geography/projection 等子包。
package geography
