package main

import "github.com/shurcooL/githubv4"

type prCommentsQuery struct {
	Repository struct {
		PullRequest struct {
			ReviewThreads struct {
				Nodes []struct {
					IsResolved githubv4.Boolean
					Comments   struct {
						Nodes []struct {
							Author struct {
								Login githubv4.String
							}
							Body      githubv4.String
							Path      githubv4.String
							Line      githubv4.Int
							URL       githubv4.URI
							CreatedAt githubv4.DateTime
						}
					} `graphql:"comments(first: 20)"`
				}
			} `graphql:"reviewThreads(first: 100)"`
		} `graphql:"pullRequest(number: $prNumber)"`
	} `graphql:"repository(owner: $owner, name: $repo)"`
}
