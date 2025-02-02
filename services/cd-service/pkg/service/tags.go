/*This file is part of kuberpult.

Kuberpult is free software: you can redistribute it and/or modify
it under the terms of the Expat(MIT) License as published by
the Free Software Foundation.

Kuberpult is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
MIT License for more details.

You should have received a copy of the MIT License
along with kuberpult. If not, see <https://directory.fsf.org/wiki/License:Expat>.

Copyright 2023 freiheit.com*/

package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/freiheit-com/kuberpult/pkg/api"
	"github.com/freiheit-com/kuberpult/services/cd-service/pkg/repository"
)

type TagsServer struct {
	Config          repository.RepositoryConfig
	OverviewService *OverviewServiceServer
}

func (s *TagsServer) GetGitTags(ctx context.Context, in *api.GetGitTagsRequest) (*api.GetGitTagsResponse, error) {
	tags, err := repository.GetTags(s.Config, "./repository_tags", ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get tags from repository: %v", err)
	}

	return &api.GetGitTagsResponse{TagData: tags}, nil
}

func (s *TagsServer) GetProductSummary(ctx context.Context, in *api.GetProductSummaryRequest) (*api.GetProductSummaryResponse, error) {
	if in.Environment == nil && in.EnvironmentGroup == nil {
		return nil, fmt.Errorf("Must have an environment or environmentGroup to get the product summary for")
	}
	if in.Environment != nil && in.EnvironmentGroup != nil {
		if *in.Environment != "" && *in.EnvironmentGroup != "" {
			return nil, fmt.Errorf("Can not have both an environment and environmentGroup to get the product summary for")
		}
	}
	if in.CommitHash == "" {
		return nil, fmt.Errorf("Must have a commit to get the product summary for")
	}
	response, err := s.OverviewService.GetOverview(ctx, &api.GetOverviewRequest{GitRevision: in.CommitHash})
	if err != nil {
		return nil, fmt.Errorf("unable to get overview for %s: %v", in.CommitHash, err)
	}

	var summaryFromEnv []api.ProductSummary
	if in.Environment != nil && *in.Environment != "" {
		for _, group := range response.EnvironmentGroups {
			for _, env := range group.Environments {
				if env.Name == *in.Environment {
					for _, app := range env.Applications {
						summaryFromEnv = append(summaryFromEnv, api.ProductSummary{App: app.Name, Version: strconv.FormatUint(app.Version, 10), Environment: *in.Environment})
					}
				}
			}
		}
		if len(summaryFromEnv) == 0 {
			return &api.GetProductSummaryResponse{}, nil
		}
		sort.Slice(summaryFromEnv, func(i, j int) bool {
			a := summaryFromEnv[i].App
			b := summaryFromEnv[j].App
			return a < b
		})
	} else {
		for _, group := range response.EnvironmentGroups {
			if *in.EnvironmentGroup == group.EnvironmentGroupName {
				for _, env := range group.Environments {
					var singleEnvSummary []api.ProductSummary
					for _, app := range env.Applications {
						singleEnvSummary = append(singleEnvSummary, api.ProductSummary{App: app.Name, Version: strconv.FormatUint(app.Version, 10), Environment: env.Name})
					}
					sort.Slice(singleEnvSummary, func(i, j int) bool {
						a := singleEnvSummary[i].App
						b := singleEnvSummary[j].App
						return a < b
					})
					summaryFromEnv = append(summaryFromEnv, singleEnvSummary...)
				}
			}
		}
		if len(summaryFromEnv) == 0 {
			return nil, nil
		}
	}

	var productVersion []*api.ProductSummary
	for _, row := range summaryFromEnv {
		for _, app := range response.Applications {
			if row.App == app.Name {
				for _, release := range app.Releases {
					if strconv.FormatUint(release.Version, 10) == row.Version {
						productVersion = append(productVersion, &api.ProductSummary{App: row.App, Version: row.Version, CommitId: release.SourceCommitId, DisplayVersion: release.DisplayVersion, Environment: row.Environment})
						break
					}
				}
			}
		}
	}
	return &api.GetProductSummaryResponse{ProductSummary: productVersion}, nil
}
