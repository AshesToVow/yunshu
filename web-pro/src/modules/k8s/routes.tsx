// @ts-nocheck
import { lazy } from "react";
import { Navigate } from '@umijs/max';
import type { RouteObject } from 'react-router';

const ClusterPage = lazy(() => import("../../pages/cluster-page").then((m) => ({ default: m.ClusterPage })));
const PodPage = lazy(() => import("../../pages/pod-page").then((m) => ({ default: m.PodPage })));

export const K8S_PLUGIN = "k8s";

export const k8sRoutes: RouteObject[] = [
  { path: "cluster", element: <Navigate to="/clusters" replace /> },
  { path: "clusters", element: <ClusterPage /> },
  { path: "pods", element: <PodPage /> },
];
