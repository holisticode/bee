// Copyright 2020 The Swarm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import "github.com/holisticode/bee/pkg/swarm"

type Server = server

type (
	BytesPostResponse     = bytesPostResponse
	ChunkAddressResponse  = chunkAddressResponse
	SocPostResponse       = socPostResponse
	FeedReferenceResponse = feedReferenceResponse
	BzzUploadResponse     = bzzUploadResponse
	TagResponse           = tagResponse
	TagRequest            = tagRequest
	ListTagsResponse      = listTagsResponse
	IsRetrievableResponse = isRetrievableResponse
	SecurityTokenResponse = securityTokenRsp
	SecurityTokenRequest  = securityTokenReq
)

var (
	InvalidContentType  = errInvalidContentType
	InvalidRequest      = errInvalidRequest
	DirectoryStoreError = errDirectoryStore
)

var (
	ContentTypeTar    = contentTypeTar
	ContentTypeHeader = contentTypeHeader
)

var (
	ErrNoResolver           = errNoResolver
	ErrInvalidNameOrAddress = errInvalidNameOrAddress
)

var (
	FeedMetadataEntryOwner = feedMetadataEntryOwner
	FeedMetadataEntryTopic = feedMetadataEntryTopic
	FeedMetadataEntryType  = feedMetadataEntryType

	SuccessWsMsg = successWsMsg
)

var (
	FileSizeBucketsKBytes = fileSizeBucketsKBytes
	ToFileSizeBucket      = toFileSizeBucket
)

func (s *Server) ResolveNameOrAddress(str string) (swarm.Address, error) {
	return s.resolveNameOrAddress(str)
}

func CalculateNumberOfChunks(contentLength int64, isEncrypted bool) int64 {
	return calculateNumberOfChunks(contentLength, isEncrypted)
}
