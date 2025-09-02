# Must-Have Changes for Next Release (v0.7.0)

This document outlines the critical changes and improvements that must be included in the next major release of the Cluster API Provider for CloudStack (CAPC).

## 🔴 Critical Bug Fixes (P0)

### 1. CloudStack User Account Deployment Issue ([#454](https://github.com/kubernetes-sigs/cluster-api-provider-cloudstack/issues/454))
- **Priority**: P0 - Critical
- **Issue**: Normal user accounts cannot deploy CAPC clusters due to CloudStack API error 432 with `listDomains` API
- **Impact**: Prevents non-admin users from using CAPC
- **Required Action**: 
  - Fix API permission requirements for normal user accounts
  - Update documentation to reflect correct CloudStack permissions
  - Add validation to prevent deployment with insufficient permissions

### 2. Load Balancer and Firewall Rules Cleanup ([#293](https://github.com/kubernetes-sigs/cluster-api-provider-cloudstack/issues/293))
- **Priority**: P0 - Critical  
- **Issue**: Load balancing and firewall rules are not deleted when CAPC cluster is destroyed
- **Impact**: Resource leakage and potential security issues
- **Required Action**:
  - Implement proper cleanup logic for port forwarding rules
  - Ensure firewall rules are removed during cluster deletion
  - Add verification tests for complete resource cleanup

## 🟡 High Priority Infrastructure Updates (P1)

### 3. CAPI v1.11.0 Compatibility ([#457](https://github.com/kubernetes-sigs/cluster-api-provider-cloudstack/issues/457))
- **Priority**: P1 - High
- **Issue**: Need to test and ensure compatibility with CAPI v1.11.0-beta.0
- **Impact**: Maintains compatibility with latest CAPI features and fixes
- **Required Action**:
  - Update dependencies to CAPI v1.11.0
  - Run comprehensive testing suite
  - Update documentation for new CAPI features
  - Address any breaking changes

### 4. Kube RBAC Proxy Conflicts ([#446](https://github.com/kubernetes-sigs/cluster-api-provider-cloudstack/issues/446))
- **Priority**: P1 - High
- **Issue**: Port 8443 conflicts between kube-rbac-proxy and CAPI diagnostic feature
- **Impact**: CAPC controller pod crashes during upgrades from v0.5.0 to v0.6.0
- **Required Action**:
  - Remove or deprecate kube-rbac-proxy server
  - Ensure CAPI diagnostic feature serves metrics properly
  - Update deployment configurations

## 🟢 Feature Enhancements (P2)

### 5. Enhanced Multi-Network Support
- **Priority**: P2 - Medium
- **Issue**: Building on v0.6.1's multiple NICs support
- **Required Action**:
  - Improve validation for multiple network configurations
  - Add comprehensive e2e tests for multi-network scenarios
  - Update documentation with best practices

### 6. Improved CloudStack SDK Integration
- **Priority**: P2 - Medium
- **Issue**: Ensure latest CloudStack SDK features are utilized
- **Required Action**:
  - Update to latest CloudStack Go SDK version
  - Implement new SDK features that benefit CAPC
  - Add error handling improvements

## 🔧 Technical Debt and Quality Improvements (P3)

### 7. Enhanced Error Handling and Logging
- **Priority**: P3 - Low
- **Required Action**:
  - Improve error messages for better troubleshooting
  - Add structured logging for better observability
  - Implement retry mechanisms for transient failures

### 8. Documentation Updates
- **Priority**: P3 - Low
- **Required Action**:
  - Update CloudStack permissions documentation ([#454](https://github.com/kubernetes-sigs/cluster-api-provider-cloudstack/issues/454))
  - Add troubleshooting guides for common issues
  - Update deployment examples with latest features

## 📋 Release Checklist

### Pre-Release Requirements
- [ ] All P0 issues resolved and tested
- [ ] CAPI v1.11.0 compatibility verified
- [ ] Comprehensive e2e test suite passes
- [ ] Security scan completed with no critical findings
- [ ] Documentation updated and reviewed

### Testing Requirements
- [ ] Unit tests pass with >90% coverage
- [ ] Integration tests pass on multiple CloudStack versions
- [ ] e2e tests pass for all supported scenarios:
  - [ ] Basic cluster creation/deletion
  - [ ] Multi-network configurations
  - [ ] VPC deployments
  - [ ] User account scenarios
  - [ ] Upgrade scenarios

### Quality Gates
- [ ] No regressions from previous release
- [ ] Performance benchmarks meet expectations
- [ ] Memory usage optimized
- [ ] Resource cleanup verified

## 🗓️ Target Timeline

| Phase | Duration | Deliverables |
|-------|----------|-------------|
| **Phase 1: Critical Fixes** | 2-3 weeks | P0 issues resolved |
| **Phase 2: CAPI Updates** | 2-3 weeks | CAPI v1.11.0 integration |
| **Phase 3: Feature Work** | 3-4 weeks | P1 features completed |
| **Phase 4: Testing & Polish** | 2-3 weeks | QA, docs, release prep |

**Total Estimated Timeline**: 9-13 weeks

## 🎯 Success Criteria

1. **Functionality**: All critical bugs fixed, no regressions
2. **Compatibility**: Full CAPI v1.11.0 support
3. **Reliability**: Improved error handling and resource cleanup
4. **Usability**: Better documentation and user experience
5. **Performance**: No performance degradation, improved efficiency where possible

## 📝 Notes

- This list is based on analysis of open issues, recent release patterns, and community feedback
- Priority levels may be adjusted based on user feedback and testing results
- Additional items may be added based on ongoing development needs
- All changes should maintain backward compatibility where possible

---

*Last Updated: September 2, 2025*
*Document Version: 1.0*