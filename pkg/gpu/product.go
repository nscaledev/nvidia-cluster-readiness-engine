// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package gpu

import "strings"

// ParseProduct extracts the GPU architecture from a GPU product name. It
// accepts both the NVIDIA GPU Feature Discovery label format, which is
// hyphen-separated (e.g. "NVIDIA-H100-80GB-HBM3"), and the ResourceSlice
// productName device attribute format used on DRA-only platforms, which is
// space-separated (e.g. "NVIDIA GB300"). It strips the "NVIDIA" prefix,
// takes the first segment before any hyphen or space, and lowercases it.
// Examples: "NVIDIA-H100-80GB-HBM3" → "h100", "NVIDIA GB300" → "gb300".
// Returns "" if the input is empty.
func ParseProduct(product string) string {
	if product == "" {
		return ""
	}
	product = strings.TrimPrefix(product, "NVIDIA-")
	product = strings.TrimPrefix(product, "NVIDIA ")
	if idx := strings.IndexAny(product, "- "); idx > 0 {
		product = product[:idx]
	}
	return strings.ToLower(product)
}
