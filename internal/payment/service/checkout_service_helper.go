package service

import "gorm.io/datatypes"

func toMapAny(m map[string]string) datatypes.JSONMap {
    res := make(datatypes.JSONMap)
    for k, v := range m {
        res[k] = v
    }
    return res
}
