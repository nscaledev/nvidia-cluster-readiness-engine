// SPDX-FileCopyrightText: Copyright (c) 2026 NVIDIA CORPORATION & AFFILIATES. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	resourcev1 "k8s.io/api/resource/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	gpuResourceSliceDriver = "gpu.nvidia.com"
	productNameAttribute   = "productName"
	gpuProductLabel        = "nvidia.com/gpu.product"

	// resourceSliceListTimeout bounds the one ResourceSlice List call so a
	// missing RBAC grant or an out-of-band (non-Helm) install degrades to
	// "log and leave nodes unlabeled" within a few seconds, rather than
	// risking an indefinite block if the cached client's informer can never
	// complete its initial sync.
	resourceSliceListTimeout = 5 * time.Second
)

// augmentGPUProductLabels sets nvidia.com/gpu.product on any node in nodes
// that lacks it, from its gpu.nvidia.com ResourceSlice productName device
// attribute (ADR-082: nscale NKS nodes carry no gpu.product label because no
// device plugin/GFD DaemonSet runs there; architecture lives in ResourceSlice
// attributes instead). Mutates the caller's in-memory Node copies only —
// never persisted back to the API server. Every existing label-based
// consumer (detectGPUArchConsistent, DetectGPUArchitecture, UniformGPUProduct,
// catalog.GPUArchFromNodeSelector, NIC detection) keeps working completely
// unchanged, because all of them only ever read the label. One ResourceSlice
// List call total, skipped entirely when every node already carries the
// label (every non-DRA cluster).
func augmentGPUProductLabels(ctx context.Context, reader client.Reader, nodes []corev1.Node) {
	needsLookup := false
	for i := range nodes {
		if nodes[i].Labels[gpuProductLabel] == "" {
			needsLookup = true
			break
		}
	}
	if !needsLookup {
		return
	}

	listCtx, cancel := context.WithTimeout(ctx, resourceSliceListTimeout)
	defer cancel()

	var slices resourcev1.ResourceSliceList
	if err := reader.List(listCtx, &slices); err != nil {
		logf.FromContext(ctx).Info("list gpu.nvidia.com resourceslices for architecture fallback failed",
			"error", err)
		return
	}

	productByNode := make(map[string]string, len(nodes))
	for _, rs := range slices.Items {
		if rs.Spec.Driver != gpuResourceSliceDriver || rs.Spec.NodeName == nil {
			continue
		}
		for _, d := range rs.Spec.Devices {
			if attr, ok := d.Attributes[productNameAttribute]; ok && attr.StringValue != nil {
				productByNode[*rs.Spec.NodeName] = *attr.StringValue
				break
			}
		}
	}

	for i := range nodes {
		if nodes[i].Labels[gpuProductLabel] != "" {
			continue
		}
		if product, ok := productByNode[nodes[i].Name]; ok {
			if nodes[i].Labels == nil {
				nodes[i].Labels = map[string]string{}
			}
			nodes[i].Labels[gpuProductLabel] = product
		}
	}
}
